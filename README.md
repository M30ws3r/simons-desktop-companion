# 🍩 시몬스 데스크톱 컴패니언

> 오늘도 타닥타닥, 시몬과 함께 일해요.

 타이핑하면 같이 방송하고, 하품하고, 설명하고, 불타오르는(!) 작은 데스크톱 시몬입니다.
한글 · Word · Excel · PowerPoint 등 어떤 프로그램을 쓰든 화면 한쪽에 떠서 함께 일합니다.
배경은 투명해서 뒤에 있는 작업 화면을 가리지 않습니다.

*A tiny desktop companion for Windows that reacts to your typing.*

<p align="center"><img src="docs/demo.gif" width="320" alt="시몬스 동작 데모"></p>

## 다운로드

**[👉 최신 버전 내려받기 (Releases)](../../releases/latest)**

`Simons-Desktop-Companion-v1.0.0-win-x64.zip`을 받아 압축을 풀고
`시몬스 데스크톱 컴패니언.exe`를 더블클릭하면 끝! 따로 설치하지 않아도 됩니다.

| | |
|---|---|
| 지원 OS | **Windows 10 / 11 (64비트) 전용** · macOS·Linux는 지원하지 않아요 |
| 설치 | 필요 없음 (실행 파일 하나) |
| 인터넷 | 사용하지 않음 |
| 크기 | 약 7.2MB |

## 이렇게 반응해요

<p align="center"><img src="docs/poses.png" width="720" alt="시몬스 포즈 모음"></p>

| 상황 | 시몬스(Simons) |
|---|---|
| 실행 | 작은 시몬, 턱 괴고 등장! |
| 타이핑 | 방송 → 하품 → 설명 → 파이어! 번갈아 바뀌어요 |
| 1분에 500타 이상을 3초 넘게 유지하는 경우 | 🔥 파이어 고정! 500타 아래로 떨어지면 풀려요 |
| 시몬스 클릭 | 도넛 생각 + 와삭바삭 소리(왁뿌볼) + vfx 🍩 |
| 15초 동안 입력 없음 | 꾸벅꾸벅 졸아요 💤 키보드 아무 키를 누르거나 클릭하면 깨어나요 |

타수는 최근 3초 동안 누른 키 수로 계산해요. Shift·Ctrl·Alt·Win·한/영 키만 단독으로 누르는 건 세지 않아요.

## 사용법

- **도넛 생각**: 작은 시몬 위에 커서를 올리면 손 모양으로 바뀌어요. 그대로 클릭!
- **옮기기**: 작은 시몬을 누른 채로 끌어서(드래그 & 드랍) 원하는 곳에 두세요.
- **메뉴**: 작은 시몬을 우클릭하면 메뉴가 열려요.
  - **크기 조절**: 커서가 십자 화살표로 바뀌면 시몬스를 누른 채로 바깥쪽으로 끌면 커지고, 중앙 쪽으로 끌면 작아져요. 크기 조절 중에 우클릭하면 취소됩니다.
  - **종료**: 프로그램을 완전히 종료합니다.
- 화면 밖으로 사라졌다면 작업 표시줄 오른쪽 **트레이 아이콘**을 눌러도 같은 메뉴가 나옵니다.
- 위치와 크기는 기억해 두었다가 다음 실행 때 그대로 열려요.
- [친칠라잼](https://github.com/M30ws3r/chinchillajam-desktop-companion)과 같이 띄워도 돼요. 설정이 따로 저장되는 구조입니다.

## 처음 실행할 때 경고가 뜬다면

코드 서명을 하지 않은 개인 제작 프로그램이라 Windows SmartScreen이
**"Windows의 PC 보호"** 창을 띄울 수 있어요(문제시 되는 건 없습니다만 windows 기본 경고입니다. 해치지 않아요...).
**추가 정보 → 실행**을 누르면 실행됩니다.

내려받은 파일이 원본 그대로인지 확인하고 싶다면 Releases 페이지에 적힌 SHA-256 값과 비교해 보세요.

```powershell
Get-FileHash .\Simons-Desktop-Companion-v1.0.0-win-x64.zip -Algorithm SHA256
```

## 개인정보 · 안전

- 작은 시몬이 타이핑에 반응하려면 **"키가 눌렸다"는 신호**가 필요해서 Windows 전역 키보드·마우스 훅을 사용해요.
- **어떤 키를 눌렀는지는 읽지도, 저장하지도, 전송하지도 않아요.** 인터넷 통신 기능 자체가 없어요.
- 저장하는 건 창 위치와 크기뿐이에요 (`%APPDATA%\SimonsCompanion\settings.json`).
- 일부 백신이 키보드 훅 때문에 오진할 수 있어요. 소스 코드는 [`app/`](app/) 폴더에 모두 공개되어 있습니다.

## 삭제

`.exe` 파일을 지우면 끝이에요. 설정까지 지우려면 `%APPDATA%\SimonsCompanion` 폴더도 삭제하세요.

## 직접 빌드하기

Go 1.21 이상이 필요해요. 외부 라이브러리는 쓰지 않아요.

```bash
cd app
go test ./...
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o "시몬스 데스크톱 컴패니언.exe" .
```

PowerShell에서는:

```powershell
cd app
$env:GOOS="windows"; $env:GOARCH="amd64"
go build -trimpath -ldflags "-H windowsgui -s -w" -o "시몬스 데스크톱 컴패니언.exe" .
```

아이콘이나 프로그램 이름을 바꿨다면 `python tools/mksyso.py icon rsrc_windows_amd64.syso`로 리소스를 다시 만드세요.
타이밍 값(500타 기준, 15초 등)은 `app/brain.go`의 `DefaultConfig()`에 있어요.

## 폴더 구성

```
app/              Windows 프로그램 소스 (Go + Win32 API)
assets/simons/    공용 리소스 (WebP 이미지, 효과음, config.json)
docs/             README용 이미지
preview.html      브라우저에서 동작 미리보기
```
