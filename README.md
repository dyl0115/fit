# fit 🎵

만물 잡동사니 CLI

> Fit 이름은 아무의미 없습니다. 쉽고 빠르게 타이핑하기 위해 Fit이라 지었습니다.
> `f`(왼손) → `i`(오른손) → `t`(왼손) — git과 같은 리듬감

---

## 설치

### 요구사항

- [Go 1.21+](https://golang.org/dl/)
- [ffmpeg](https://ffmpeg.org/download.html) (라디오 + 음악 재생에 필요)

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

```bash
fit radio list           # 등록된 스테이션 목록
fit radio play <번호>    # 라디오 재생
fit radio stop           # 재생 중지
fit radio status         # 현재 재생 중인 라디오 확인
fit radio add <이름> <url> [장르]  # 스테이션 추가
fit radio remove <번호>  # 스테이션 삭제
```

#### 기본 제공 스테이션

| # | 이름 | 장르 |
|---|------|------|
| 1 | Radio Paradise | Rock/Eclectic |
| 2 | SomaFM Groove Salad | Ambient/Chill |
| 3 | SomaFM Indie Pop Rocks | Indie Pop |
| 4 | 1.FM Absolute Top 40 | Pop/Top40 |
| 5 | EBS FM | Korean/EBS |
| 6 | TBS FM | Korean/TBS |
| 7 | CBS 음악FM | Korean/CBS |
| 8 | 국악방송 | Korean/국악 |
| 9 | KBS 1FM | Korean/KBS |
| 10 | KBS 2FM (Cool FM) | Korean/KBS |
| 11 | KBS 1라디오 | Korean/KBS |
| 12 | KBS 2라디오 | Korean/KBS |
| 13 | MBC FM4U | Korean/MBC |
| 14 | MBC 표준FM | Korean/MBC |
| 15 | SBS 파워FM | Korean/SBS |
| 16 | SBS 러브FM | Korean/SBS |

---

## 데이터 저장 위치

| 항목 | 경로 |
|------|------|
| 라디오 스테이션 목록 | `%AppData%\fit\radio_stations.json` |
| 현재 재생 상태 | `%TEMP%\fit_music_state.json` |
| Music 프로세스 PID | `%TEMP%\fit_music.pid` |
| Radio 프로세스 PID | `%TEMP%\fit_radio.pid` |

---

## 기술 스택

- **Go** + **Cobra** - CLI 프레임워크
- **hajimehoshi/oto** - 로컬 음악 오디오 출력
- **hajimehoshi/go-mp3** - MP3 디코딩
- **ffplay** (ffmpeg) - 라디오 스트리밍 재생

---

## License

MIT
