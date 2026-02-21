package music

import (
	"encoding/json"
	"fmt"
	"math/rand"
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

func stateFilePath() string { return filepath.Join(os.TempDir(), "fit_music_state.json") }
func pidFilePath() string   { return filepath.Join(os.TempDir(), "fit_music.pid") }
func skipFilePath() string  { return filepath.Join(os.TempDir(), "fit_music.skip") }
func queueFilePath() string { return filepath.Join(os.TempDir(), "fit_music_queue.json") }

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

func readState() (*MusicState, error) {
	data, err := os.ReadFile(stateFilePath())
	if err != nil {
		return nil, err
	}
	var state MusicState
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

var MusicCmd = &cobra.Command{
	Use:   "music",
	Short: "음악 관련 기능",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "백그라운드에서 Music 폴더 랜덤 재생",
	Run:   runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "재생 중지",
	Run:   runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "현재 재생 중인 곡 확인",
	Run:   runStatus,
}

var skipCmd = &cobra.Command{
	Use:   "skip",
	Short: "다음 곡으로 넘기기",
	Run:   runSkip,
}

var internalRunCmd = &cobra.Command{
	Use:    "_run",
	Hidden: true,
	Run:    runInternal,
}

func init() {
	MusicCmd.AddCommand(startCmd)
	MusicCmd.AddCommand(stopCmd)
	MusicCmd.AddCommand(statusCmd)
	MusicCmd.AddCommand(skipCmd)
	MusicCmd.AddCommand(internalRunCmd)
}

// ───────────────────────────── 커맨드 핸들러 ─────────────────────────────

func runStart(cmd *cobra.Command, args []string) {
	if isRunning() {
		fmt.Println("이미 재생 중입니다. `fit music status` 로 확인하세요.")
		return
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "실행 파일 경로를 찾을 수 없습니다:", err)
		os.Exit(1)
	}

	proc := exec.Command(exe, "music", "_run")
	proc.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	proc.Stdout = nil
	proc.Stderr = nil
	proc.Stdin = nil

	if err := proc.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "백그라운드 실행 실패:", err)
		os.Exit(1)
	}

	writePID(proc.Process.Pid)
	fmt.Printf("🎵 음악 재생 시작! (PID: %d)\n", proc.Process.Pid)
	fmt.Println("  fit music status  - 현재 곡 확인")
	fmt.Println("  fit music skip    - 다음 곡")
	fmt.Println("  fit music stop    - 정지")
}

func runStop(cmd *cobra.Command, args []string) {
	pid, err := readPID()
	if err != nil {
		fmt.Println("재생 중인 음악이 없습니다.")
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
	os.Remove(skipFilePath())
	os.Remove(queueFilePath())
	fmt.Println("⏹  재생을 중지했습니다.")
}

func runStatus(cmd *cobra.Command, args []string) {
	if !isRunning() {
		fmt.Println("재생 중인 음악이 없습니다. `fit music start` 로 시작하세요.")
		return
	}
	state, err := readState()
	if err != nil {
		fmt.Println("상태를 읽을 수 없습니다.")
		return
	}
	fmt.Printf("🎵 %s\n", state.CurrentTrack)
	fmt.Printf("   재생 시작: %s\n", state.StartedAt)
}

func runSkip(cmd *cobra.Command, args []string) {
	if !isRunning() {
		fmt.Println("재생 중인 음악이 없습니다.")
		return
	}
	os.WriteFile(skipFilePath(), []byte("skip"), 0644)
	fmt.Println("⏭  다음 곡으로 넘깁니다.")
}

// ───────────────────────────── 재생 루프 (_run) ─────────────────────────────

func runInternal(cmd *cobra.Command, args []string) {
	musicDir, err := getMusicDir()
	if err != nil {
		os.Exit(1)
	}
	files, err := collectMP3s(musicDir)
	if err != nil || len(files) == 0 {
		os.Exit(1)
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
		for _, path := range files {
			playWithFFplay(path)
		}
	}
}

func playWithFFplay(path string) {
	writeState(filepath.Base(path), "local")

	proc := exec.Command("ffplay", "-nodisp", "-loglevel", "quiet", "-autoexit", path)
	proc.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	proc.Stdout = nil
	proc.Stderr = nil
	proc.Stdin = nil

	if err := proc.Start(); err != nil {
		return
	}

	// ffplay 프로세스 PID를 별도 파일에 저장 (skip용)
	ffplayPidPath := filepath.Join(os.TempDir(), "fit_ffplay.pid")
	os.WriteFile(ffplayPidPath, []byte(strconv.Itoa(proc.Process.Pid)), 0644)
	defer os.Remove(ffplayPidPath)

	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if _, err := os.Stat(skipFilePath()); err == nil {
				os.Remove(skipFilePath())
				proc.Process.Kill()
				return
			}
		}
	}
}

// ───────────────────────────── 유틸 ─────────────────────────────

func getMusicDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	musicDir := filepath.Join(home, "Music")
	if _, err := os.Stat(musicDir); err != nil {
		return "", err
	}
	return musicDir, nil
}

func collectMP3s(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".mp3" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
