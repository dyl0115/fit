# fit 🎵

만물 잡동사니 CLI

> Fit 이름은 큰 의미 없습니다. 쉽고 빠르게 타이핑하기 위해 Fit이라 지었습니다.

---

## ⚠️ 외부 프로그램 의존성

**이거 먼저 설치 안 하면 radio / youtube 기능이 작동하지 않습니다.**

### 1. ffmpeg (ffplay 포함)

라디오 스트리밍 및 유튜브 오디오 재생에 필요합니다.

- 다운로드: https://ffmpeg.org/download.html
- Windows 추천: https://www.gyan.dev/ffmpeg/builds/ 에서 `ffmpeg-release-full.7z` 다운로드
- 압축 해제 후 `bin` 폴더 안의 `ffplay.exe`, `ffmpeg.exe`를 PATH에 등록

설치 확인:
```bash
ffplay -version
```

### 2. yt-dlp

유튜브 오디오 스트리밍에 필요합니다.

```bash
winget install yt-dlp
```

또는 https://github.com/yt-dlp/yt-dlp/releases 에서 직접 다운로드 후 PATH 등록

설치 확인:
```bash
yt-dlp --version
```

---

## 설치

### 요구사항

- [Go 1.21+](https://golang.org/dl/)
- ffmpeg (위 참고)
- yt-dlp (위 참고)

### 빌드

```bash
git clone https://github.com/dyl01/fit
cd fit
go mod tidy
go build -o fit.exe .
```

빌드된 `fit.exe`를 PATH에 등록하면 어디서든 사용 가능합니다.

---

## 기능

### 🎵 music - 로컬 음악 재생

Windows `~/Music` 폴더의 mp3 파일을 랜덤 셔플로 백그라운드 재생합니다.

```bash
fit music start    # 백그라운드 재생 시작
fit music stop     # 재생 중지
fit music status   # 현재 재생 중인 곡 확인
fit music skip     # 다음 곡으로 넘기기
```

### 📻 radio - 인터넷 라디오 스트리밍

국내외 라디오를 스트리밍합니다. KBS, MBC, SBS는 동적 API를 통해 스트리밍 URL을 자동으로 가져옵니다.

> **필요:** ffplay

```bash
fit radio list                         # 등록된 스테이션 목록
fit radio play <번호>                  # 라디오 재생
fit radio stop                         # 재생 중지
fit radio status                       # 현재 재생 중인 라디오 확인
fit radio add <이름> <url> [장르]      # 스테이션 추가
fit radio remove <번호>                # 스테이션 삭제
```

#### 기본 제공 스테이션

| # | 이름 | 장르 |
|---|------|------|
| 1 | EBS FM | Korean/EBS |
| 2 | TBS FM | Korean/TBS |
| 3 | CBS 음악FM | Korean/CBS |
| 4 | 국악방송 | Korean/국악 |
| 5 | KBS 1FM | Korean/KBS |
| 6 | KBS 2FM (Cool FM) | Korean/KBS |
| 7 | KBS 1라디오 | Korean/KBS |
| 8 | KBS 2라디오 | Korean/KBS |
| 9 | MBC FM4U | Korean/MBC |
| 10 | MBC 표준FM | Korean/MBC |
| 11 | SBS 파워FM | Korean/SBS |
| 12 | SBS 러브FM | Korean/SBS |

### 🎬 youtube - 유튜브 오디오 재생

유튜브 영상의 오디오만 추출해 백그라운드로 재생합니다. ASMR, 백색소음 등 장시간 틀어놓기 적합합니다.

> **필요:** yt-dlp + ffplay
> yt-dlp가 오디오 스트림을 받아 ffplay로 파이프 전달하는 방식으로 동작합니다.

```bash
fit youtube list                        # 등록된 유튜브 목록
fit youtube play <번호>                 # 오디오 재생
fit youtube stop                        # 재생 중지
fit youtube status                      # 현재 재생 중인 항목 확인
fit youtube add <이름> <url> [장르]     # 유튜브 주소 추가
fit youtube remove <번호>               # 유튜브 주소 삭제
```

#### 기본 제공 목록

| # | 이름 | 장르 |
|---|------|------|
| 1 | 빗소리 10시간 | ASMR |
| 2 | 심해소리 8시간 | ASMR |

---

## 데이터 저장 위치

| 항목 | 경로 |
|------|------|
| 라디오 스테이션 목록 | `%AppData%\fit\radio_stations.json` |
| 유튜브 목록 | `%AppData%\fit\youtube_urls.json` |
| Radio 재생 상태 | `%TEMP%\fit_music_state.json` |
| Youtube 재생 상태 | `%TEMP%\fit_youtube_state.json` |
| Radio 프로세스 PID | `%TEMP%\fit_radio.pid` |
| Youtube 프로세스 PID | `%TEMP%\fit_youtube.pid` |

---

## 기술 스택

- **Go** + **Cobra** - CLI 프레임워크
- **hajimehoshi/oto** - 로컬 음악 오디오 출력
- **hajimehoshi/go-mp3** - MP3 디코딩
- **ffplay** (ffmpeg) - 라디오/유튜브 오디오 재생
- **yt-dlp** - 유튜브 오디오 스트림 추출

---

## License

MIT
