package radio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func stationsFilePath() string {
	configDir, _ := os.UserConfigDir()
	dir := filepath.Join(configDir, "fit")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "radio_stations.json")
}

func pidFilePath() string   { return filepath.Join(os.TempDir(), "fit_radio.pid") }
func stateFilePath() string { return filepath.Join(os.TempDir(), "fit_music_state.json") }

// ───────────────────────────── 스테이션 관리 ─────────────────────────────

// Type: "static" | "kbs" | "mbc" | "sbs"
type Station struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Genre string `json:"genre"`
	Type  string `json:"type"`
}

var defaultStations = []Station{
	// 한국 정적 m3u8
	{Name: "EBS FM", URL: "https://ebsonair.ebs.co.kr/fmradiofamilypc/familypc1m/playlist.m3u8", Genre: "Korean/EBS", Type: "static"},
	{Name: "TBS FM", URL: "https://cdnfm.tbs.seoul.kr/tbs/_definst_/tbs_fm_web_360.smil/playlist.m3u8", Genre: "Korean/TBS", Type: "static"},
	{Name: "CBS 음악FM", URL: "https://aac.cbs.co.kr/cbs981/_definst_/cbs981.stream/playlist.m3u8", Genre: "Korean/CBS", Type: "static"},
	{Name: "국악방송", URL: "https://mgugaklive.nowcdn.co.kr/gugakradio/gugakradio.stream/playlist.m3u8", Genre: "Korean/국악", Type: "static"},
	// KBS (동적 API)
	{Name: "KBS 1FM", URL: "https://cfpwwwapi.kbs.co.kr/api/v1/landing/live/channel_code/24", Genre: "Korean/KBS", Type: "kbs"},
	{Name: "KBS 2FM (Cool FM)", URL: "https://cfpwwwapi.kbs.co.kr/api/v1/landing/live/channel_code/25", Genre: "Korean/KBS", Type: "kbs"},
	{Name: "KBS 1라디오", URL: "https://cfpwwwapi.kbs.co.kr/api/v1/landing/live/channel_code/21", Genre: "Korean/KBS", Type: "kbs"},
	{Name: "KBS 2라디오", URL: "https://cfpwwwapi.kbs.co.kr/api/v1/landing/live/channel_code/22", Genre: "Korean/KBS", Type: "kbs"},
	// MBC (동적 API)
	{Name: "MBC FM4U", URL: "https://sminiplay.imbc.com/aacplay.ashx?agent=webapp&channel=mfm", Genre: "Korean/MBC", Type: "mbc"},
	{Name: "MBC 표준FM", URL: "https://sminiplay.imbc.com/aacplay.ashx?agent=webapp&channel=sfm", Genre: "Korean/MBC", Type: "mbc"},
	// SBS (동적 API)
	{Name: "SBS 파워FM", URL: "https://apis.sbs.co.kr/play-api/1.0/livestream/powerpc/powerfm?protocol=hls&ssl=Y", Genre: "Korean/SBS", Type: "sbs"},
	{Name: "SBS 러브FM", URL: "https://apis.sbs.co.kr/play-api/1.0/livestream/lovepc/lovefm?protocol=hls&ssl=Y", Genre: "Korean/SBS", Type: "sbs"},
}

func loadStations() []Station {
	data, err := os.ReadFile(stationsFilePath())
	if err != nil {
		saveStations(defaultStations)
		return defaultStations
	}
	var stations []Station
	if err := json.Unmarshal(data, &stations); err != nil {
		return defaultStations
	}
	return stations
}

func saveStations(stations []Station) {
	data, _ := json.MarshalIndent(stations, "", "  ")
	os.WriteFile(stationsFilePath(), data, 0644)
}

// ───────────────────────────── 동적 URL 추출 ─────────────────────────────

func resolveStreamURL(s Station) (string, error) {
	switch s.Type {
	case "static":
		return s.URL, nil
	case "kbs":
		return resolveKBS(s.URL)
	case "mbc":
		return resolvePlainText(s.URL)
	case "sbs":
		return resolvePlainText(s.URL)
	default:
		return s.URL, nil
	}
}

func resolveKBS(apiURL string) (string, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		ChannelItem []struct {
			ServiceURL string `json:"service_url"`
			Bitrate    string `json:"bitrate"`
		} `json:"channel_item"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("KBS API 파싱 실패: %w", err)
	}
	if len(result.ChannelItem) == 0 {
		return "", fmt.Errorf("KBS API: 스트리밍 URL을 찾을 수 없습니다")
	}
	// 128Kbps 항목 우선 선택
	for _, item := range result.ChannelItem {
		if item.Bitrate == "128Kbps" && item.ServiceURL != "" {
			return item.ServiceURL, nil
		}
	}
	return result.ChannelItem[0].ServiceURL, nil
}

func resolvePlainText(apiURL string) (string, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	url := strings.TrimSpace(string(body))
	if url == "" {
		return "", fmt.Errorf("스트리밍 URL을 가져올 수 없습니다")
	}
	return url, nil
}

// ───────────────────────────── 상태 관리 ─────────────────────────────

type MusicState struct {
	CurrentTrack string `json:"current_track"`
	StartedAt    string `json:"started_at"`
	Mode         string `json:"mode"`
}

func writeState(track, mode string) {
	state := MusicState{
		CurrentTrack: track,
		StartedAt:    time.Now().Format("15:04:05"),
		Mode:         mode,
	}
	data, _ := json.Marshal(state)
	os.WriteFile(stateFilePath(), data, 0644)
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

var RadioCmd = &cobra.Command{
	Use:   "radio",
	Short: "인터넷 라디오 스트리밍",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "등록된 라디오 목록",
	Run:   runList,
}

var playCmd = &cobra.Command{
	Use:   "play <번호>",
	Short: "라디오 재생 (번호는 fit radio list 에서 확인)",
	Args:  cobra.ExactArgs(1),
	Run:   runPlay,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "라디오 중지",
	Run:   runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "현재 재생 중인 라디오 확인",
	Run:   runStatus,
}

var addCmd = &cobra.Command{
	Use:   "add <이름> <url> [장르]",
	Short: "라디오 스테이션 추가 (static으로 등록됨)",
	Args:  cobra.RangeArgs(2, 3),
	Run:   runAdd,
}

var removeCmd = &cobra.Command{
	Use:   "remove <번호>",
	Short: "라디오 스테이션 삭제",
	Args:  cobra.ExactArgs(1),
	Run:   runRemove,
}

func init() {
	RadioCmd.AddCommand(listCmd)
	RadioCmd.AddCommand(playCmd)
	RadioCmd.AddCommand(stopCmd)
	RadioCmd.AddCommand(statusCmd)
	RadioCmd.AddCommand(addCmd)
	RadioCmd.AddCommand(removeCmd)
}

// ───────────────────────────── 커맨드 핸들러 ─────────────────────────────

func runList(cmd *cobra.Command, args []string) {
	stations := loadStations()
	fmt.Println("📻 등록된 라디오 스테이션:")
	fmt.Println()
	for i, s := range stations {
		fmt.Printf("  %2d. %-30s [%s]\n", i+1, s.Name, s.Genre)
	}
	fmt.Println()
	fmt.Println("  fit radio play <번호>  으로 재생하세요.")
}

func runPlay(cmd *cobra.Command, args []string) {
	if isRunning() {
		fmt.Println("이미 라디오가 재생 중입니다. `fit radio stop` 으로 먼저 중지하세요.")
		return
	}

	num, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("번호를 입력하세요. 예: fit radio play 1")
		return
	}

	stations := loadStations()
	if num < 1 || num > len(stations) {
		fmt.Printf("번호 범위를 벗어났습니다. 1~%d 사이로 입력하세요.\n", len(stations))
		return
	}

	found := stations[num-1]

	// 동적 URL이면 API 먼저 호출
	fmt.Printf("📡 %s 연결 중...\n", found.Name)
	streamURL, err := resolveStreamURL(found)
	if err != nil {
		fmt.Fprintln(os.Stderr, "스트리밍 URL 가져오기 실패:", err)
		return
	}

	proc := exec.Command("ffplay", "-nodisp", "-loglevel", "quiet", streamURL)
	proc.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	proc.Stdout = nil
	proc.Stderr = nil
	proc.Stdin = nil

	if err := proc.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "ffplay 실행 실패. ffmpeg가 설치되어 있는지 확인하세요:", err)
		return
	}

	writePID(proc.Process.Pid)
	writeState(found.Name, "radio")

	fmt.Printf("📻 %s 재생 중! (PID: %d)\n", found.Name, proc.Process.Pid)
	fmt.Printf("   장르: %s\n", found.Genre)
	fmt.Println("  fit radio stop    - 정지")
	fmt.Println("  fit radio status  - 현재 확인")
}

func runStop(cmd *cobra.Command, args []string) {
	pid, err := readPID()
	if err != nil {
		fmt.Println("재생 중인 라디오가 없습니다.")
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("프로세스를 찾을 수 없습니다.")
		return
	}
	if err := proc.Kill(); err != nil {
		fmt.Println("종료 실패:", err)
		return
	}
	os.Remove(pidFilePath())
	os.Remove(stateFilePath())
	fmt.Println("⏹  라디오를 중지했습니다.")
}

func runStatus(cmd *cobra.Command, args []string) {
	if !isRunning() {
		fmt.Println("재생 중인 라디오가 없습니다. `fit radio list` 로 목록을 확인하세요.")
		return
	}
	data, err := os.ReadFile(stateFilePath())
	if err != nil {
		fmt.Println("상태를 읽을 수 없습니다.")
		return
	}
	var state MusicState
	json.Unmarshal(data, &state)
	fmt.Printf("📻 %s\n", state.CurrentTrack)
	fmt.Printf("   재생 시작: %s\n", state.StartedAt)
}

func runAdd(cmd *cobra.Command, args []string) {
	stations := loadStations()
	genre := "기타"
	if len(args) == 3 {
		genre = args[2]
	}
	stations = append(stations, Station{Name: args[0], URL: args[1], Genre: genre, Type: "static"})
	saveStations(stations)
	fmt.Printf("✅ '%s' 추가 완료!\n", args[0])
}

func runRemove(cmd *cobra.Command, args []string) {
	num, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("번호를 입력하세요. 예: fit radio remove 1")
		return
	}
	stations := loadStations()
	if num < 1 || num > len(stations) {
		fmt.Printf("번호 범위를 벗어났습니다. 1~%d 사이로 입력하세요.\n", len(stations))
		return
	}
	removed := stations[num-1]
	stations = append(stations[:num-1], stations[num:]...)
	saveStations(stations)
	fmt.Printf("🗑  '%s' 삭제 완료!\n", removed.Name)
}
