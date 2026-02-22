package youtube

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// ───────────────────────────── 파일 경로 ─────────────────────────────
func urlsFilePath() string {
	configDir, _ := os.UserConfigDir()
	dir := filepath.Join(configDir, "fit")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "youtube_urls.json")
}
func stateFilePath() string { return filepath.Join(os.TempDir(), "fit_youtube_state.json") }
func pidFilePath() string   { return filepath.Join(os.TempDir(), "fit_youtube.pid") }

// ───────────────────────────── 유튜브 주소 관리 ─────────────────────────────
type YoutubeUrl struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Genre string `json:"genre"`
}

var defaultUrls = []YoutubeUrl{
	{Name: "빗소리 10시간", URL: "https://www.youtube.com/watch?v=lQ0fS2meTYQ", Genre: "ASMR"},
	{Name: "심해소리 8시간", URL: "https://www.youtube.com/watch?v=ZDurqoIw-Xs", Genre: "ASMR"},
}

func loadYoutubeUrls() []YoutubeUrl {
	data, err := os.ReadFile(urlsFilePath())
	if err != nil {
		saveUrls(defaultUrls)
		return defaultUrls
	}
	var urls []YoutubeUrl
	if err := json.Unmarshal(data, &urls); err != nil {
		return defaultUrls
	}
	return urls
}

func saveUrls(urls []YoutubeUrl) {
	data, _ := json.MarshalIndent(urls, "", "  ")
	os.WriteFile(urlsFilePath(), data, 0644)
}

// ───────────────────────────── 상태 관리 ─────────────────────────────

type YoutubeState struct {
	CurrentTrack string `json:"current_track"`
	StartedAt    string `json:"started_at"`
	URL          string `json:"url"`
	Genre        string `json:"genre"`
}

func writeState(track, url, genre string) {
	state := YoutubeState{
		CurrentTrack: track,
		StartedAt:    time.Now().Format("15:04:05"),
		URL:          url,
		Genre:        genre,
	}
	data, _ := json.Marshal(state)
	os.WriteFile(stateFilePath(), data, 0644)
}

func readState() (*YoutubeState, error) {
	data, err := os.ReadFile(stateFilePath())
	if err != nil {
		return nil, err
	}
	var state YoutubeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func writePID(pid int) {
	os.WriteFile(pidFilePath(), []byte(strconv.Itoa(pid)), 0644)
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func isRunning() bool {
	pid, err := readPID()
	if err != nil {
		return false
	}
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(handle)
	return true
}

// ───────────────────────────── 커맨드 정의 ─────────────────────────────

var YoutubeCmd = &cobra.Command{
	Use:   "youtube",
	Short: "유튜브 오디오 재생",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "등록된 유튜브 목록",
	Run:   runList,
}

var playCmd = &cobra.Command{
	Use:   "play <번호>",
	Short: "유튜브 오디오 재생 (번호는 fit youtube list 에서 확인)",
	Args:  cobra.ExactArgs(1),
	Run:   runPlay,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "재생 중지",
	Run:   runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "현재 재생 중인 항목 확인",
	Run:   runStatus,
}

var addCmd = &cobra.Command{
	Use:   "add <이름> <url> [장르]",
	Short: "유튜브 주소 추가",
	Args:  cobra.RangeArgs(2, 3),
	Run:   runAdd,
}

var removeCmd = &cobra.Command{
	Use:   "remove <번호>",
	Short: "유튜브 주소 삭제",
	Args:  cobra.ExactArgs(1),
	Run:   runRemove,
}

func init() {
	YoutubeCmd.AddCommand(listCmd)
	YoutubeCmd.AddCommand(playCmd)
	YoutubeCmd.AddCommand(stopCmd)
	YoutubeCmd.AddCommand(statusCmd)
	YoutubeCmd.AddCommand(addCmd)
	YoutubeCmd.AddCommand(removeCmd)
}

// ───────────────────────────── 커맨드 핸들러 ─────────────────────────────

func runList(cmd *cobra.Command, args []string) {
	urls := loadYoutubeUrls()
	fmt.Println("🎵 등록된 유튜브 목록:")
	fmt.Println()
	for i, u := range urls {
		fmt.Printf("  %2d. %-30s [%s]\n", i+1, u.Name, u.Genre)
	}
	fmt.Println()
	fmt.Println("  fit youtube play <번호>  으로 재생하세요.")
}

func runPlay(cmd *cobra.Command, args []string) {
	if isRunning() {
		fmt.Println("이미 재생 중입니다. `fit youtube stop` 으로 먼저 중지하세요.")
		return
	}

	num, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("번호를 입력하세요. 예: fit youtube play 1")
		return
	}

	urls := loadYoutubeUrls()
	if num < 1 || num > len(urls) {
		fmt.Printf("번호 범위를 벗어났습니다. 1~%d 사이로 입력하세요.\n", len(urls))
		return
	}

	found := urls[num-1]

	fmt.Printf("🔍 %s 스트리밍 준비 중...\n", found.Name)

	// yt-dlp로 오디오 스트림 → ffplay 파이프
	// yt-dlp -f 140 -o - <url> | ffplay -nodisp -loglevel quiet -i pipe:0
	ytdlp := exec.Command("yt-dlp", "-f", "140", "-o", "-", found.URL)
	ffplay := exec.Command("ffplay", "-nodisp", "-loglevel", "quiet", "-i", "pipe:0")

	// yt-dlp stdout → ffplay stdin 파이프 연결
	pipe, err := ytdlp.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "❌ 파이프 생성 실패:", err)
		return
	}
	ffplay.Stdin = pipe

	ytdlp.Stderr = nil
	ffplay.Stdout = nil
	ffplay.Stderr = nil

	// 백그라운드 실행을 위해 새 프로세스 그룹
	ytdlp.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	ffplay.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}

	if err := ytdlp.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "❌ yt-dlp 실행 실패. yt-dlp가 설치되어 있는지 확인하세요:", err)
		return
	}
	if err := ffplay.Start(); err != nil {
		ytdlp.Process.Kill()
		fmt.Fprintln(os.Stderr, "❌ ffplay 실행 실패:", err)
		return
	}

	// 두 PID를 같이 저장 (stop 시 둘 다 종료)
	pids := fmt.Sprintf("%d %d", ytdlp.Process.Pid, ffplay.Process.Pid)
	os.WriteFile(pidFilePath(), []byte(pids), 0644)
	writeState(found.Name, found.URL, found.Genre)

	fmt.Printf("▶  %s 재생 중!\n", found.Name)
	fmt.Printf("   장르: %s\n", found.Genre)
	fmt.Println("  fit youtube stop    - 정지")
	fmt.Println("  fit youtube status  - 현재 확인")
}

func runStop(cmd *cobra.Command, args []string) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		fmt.Println("재생 중인 항목이 없습니다.")
		return
	}

	// 두 PID 파싱해서 둘 다 종료
	parts := strings.Fields(string(data))
	for _, p := range parts {
		pid, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		proc.Kill()
	}

	os.Remove(pidFilePath())
	os.Remove(stateFilePath())
	fmt.Println("⏹  재생을 중지했습니다.")
}

func readPIDs() bool {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return false
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return false
	}
	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(handle)
	return true
}

func runStatus(cmd *cobra.Command, args []string) {
	if !readPIDs() {
		fmt.Println("재생 중인 항목이 없습니다. `fit youtube list` 로 목록을 확인하세요.")
		return
	}
	state, err := readState()
	if err != nil {
		fmt.Println("상태를 읽을 수 없습니다.")
		return
	}
	fmt.Printf("▶  %s\n", state.CurrentTrack)
	fmt.Printf("   장르: %s\n", state.Genre)
	fmt.Printf("   재생 시작: %s\n", state.StartedAt)
}

func runAdd(cmd *cobra.Command, args []string) {
	urls := loadYoutubeUrls()
	genre := "기타"
	if len(args) == 3 {
		genre = args[2]
	}
	urls = append(urls, YoutubeUrl{Name: args[0], URL: args[1], Genre: genre})
	saveUrls(urls)
	fmt.Printf("✅ '%s' 추가 완료!\n", args[0])
}

func runRemove(cmd *cobra.Command, args []string) {
	num, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("번호를 입력하세요. 예: fit youtube remove 1")
		return
	}
	urls := loadYoutubeUrls()
	if num < 1 || num > len(urls) {
		fmt.Printf("번호 범위를 벗어났습니다. 1~%d 사이로 입력하세요.\n", len(urls))
		return
	}
	removed := urls[num-1]
	urls = append(urls[:num-1], urls[num:]...)
	saveUrls(urls)
	fmt.Printf("🗑  '%s' 삭제 완료!\n", removed.Name)
}
