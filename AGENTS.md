# AGENTS.md — WinCryptSSHAgent (форк tantra35)

Инструкции для AI-агентов и шпаргалка по неочевидным местам проекта.
Форк [buptczq/WinCryptSSHAgent](https://github.com/buptczq/WinCryptSSHAgent):
SSH-агент для Windows на Windows CryptoAPI (Go, systray на lxn/walk).

## Сборка

```cmd
build.cmd
```

- Скрипт использует `%MYTOOLSPATH%\go-1.23.x` и `%MYLIBSPATH%\Golang` (GOPATH).
- `build-embed-rsrc.cmd` перед сборкой перегенерирует `rsrc.syso` (манифест + иконка) через `rsrc`.
- Ключевой флаг: `-ldflags="-H windowsgui"` — **GUI-подсистема**. Следствия:
  - stdout/stderr не существуют, вывод теряется молча;
  - `log.Fatal` / паники невидимы — для диагностики нужен файловый лог;
  - консольный `go build` (без флага) даёт другое поведение при запуске с ярлыка.
- Быстрая проверочная сборка из bash: `export PATH="/c/Tools/go-1.25.x/bin:$PATH" && go build -ldflags="-H windowsgui" -o test.exe .` (go.mod требует Go ≥ 1.23).

## Запуск

```
WinCryptSSHAgent.exe [keys.ppk ...] [флаги]
```

- Позиционные аргументы — пути к PuTTY `.ppk` ключам (незашифрованные; зашифрованные молча пропускаются).
- Без конфига используется дефолтный набор из 6 сокетов (wsl, vsock, cygwin, named-pipe, pageant, xshell).
- Дефолтный конфиг: `WinCryptSSHAgent.yaml` **рядом с exe** (не в cwd). Явно заданный `--config` обязан существовать, отсутствие дефолтного — не ошибка (fallback на дефолтные сокеты).
- Флаги: `-v/-vvv` (verbose/debug), `-i` (установка Hyper-V сервиса), `--disable-capi`, `--disable-pin-cache`, `--allow-multiple`, `--socket type=name|path`, `--no-socket name`, `--list-sockets`.
- Single-instance: Windows mutex `Local\WinCryptSSHAgent.Mutex`; второй экземпляр показывает MsgBox и завершается. `--allow-multiple` снимает запрет.

## Архитектура

- `main.go` — CLI (cobra), сборка приложений из конфига, systray, `acquireInstanceMutex`.
- `config/` — viper-конфиг: модель `Socket{type,name,path,keys,notify}`, 6 типов сокетов (`named-pipe`, `pageant`, `cygwin`, `wsl`, `xshell`, `vsock`), `Resolve()` (merge CLI-флагов поверх файла), `DefaultSockets()`.
- `app/` — реализации транспортов; единый интерфейс `app.App{Name, Run, Menu}`, фабрика `NewSocket`. Shutdown-паттерн везде одинаковый: горутина `l.Close()` по `ctx.Done()` + `errors.Is(err, net.ErrClosed)` → `wg.Wait()`. В pipe.go дополнительно проверяется `winio.ErrPipeListenerClosed`.
- `sshagent/` — бэкенды ключей: `CAPIAgent` (Windows Certificate Store / смарт-карты), `KeyRingAgent` (ppk в памяти), `HVAgent` (Hyper-V), `WrappedAgent` (композиция: Sign/SignWithFlags пробует всех по очереди).
- `utils/` — утилиты (уведомления, pageant-протокол, WSL2, hyper-v, окно/clipboard).

`agent.Agent.Sign(key ssh.PublicKey, data []byte)`: `key` — публичный ключ-селектор (клиент выбирает из списка `List()`), реализация ищет соответствующий приватный и подписывает `data`.

## Грабли (важно!)

### cobra mousetrap — «процесс умирает через ровно 5 секунд» (уже исправлено, не рецидивировать)

CLI на cobra тянет зависимость `inconshreveable/mousetrap`. На Windows при запуске **двойным кликом / из ярлыка** (родительский процесс — `explorer.exe`) cobra печатает help (в GUI-бинарнике — в никуда), делает `time.Sleep(5 * time.Second)` и `os.Exit(1)`. Симптом: агент молча умирает через ровно 5 секунд при запуске с ярлыка, но нормально работает из консоли. Нет ни паники, ни WER-записи, ни Defender-детекта — чистый exit.

Фикс (в main.go): `cobra.MousetrapHelpText = ""` — ловушка отключена, GUI-приложение имеет право запускаться двойным кликом. При рефакторинге CLI не потерять.

Как ловить такие «тихие смерти»: текстовый лог в user-writable каталог (`%LOCALAPPDATA%\<app>\`), heartbeat-горутина раз в секунду + `runtime.Stack(all=true)` дампа на 2-й и 4-й секунде. Windows Event Log и WER при чистом `os.Exit` молчат; гипотеза «антивирус» проверять последней — сначала дамп стеков.

### Запуск из консоли ≠ запуск с ярлыка

Поведение GUI-приложения различается в зависимости от родительского процесса (explorer.exe / cmd / bash). Все тесты «запускается/не запускается» проводить обоими способами.

### Копирование бинаря поверх работающего экземпляра

`cp` в занятый exe падает с «Device or resource busy» — копия не обновляется, а ошибка легко пропустить. Перед деплоем: `taskkill /IM WinCryptSSHAgent.exe /F`. После копирования сверять md5.

### Shutdown и тайминги

Задержка при выходе устранена (раньше была ~5с — cap в main.go как safety net). Не возвращать блокирующие `Accept` без close-on-ctx и не путать `io.ErrClosedPipe` с `*net.OpError` (`errors.Is(err, net.ErrClosed)`).

### Совместимость ssh-клиентов

- msys-ssh (Git/usr/bin/ssh, OpenSSH 10.x) не умеет named pipe — cygwin-сокет тестировать им, named pipe — нативным `C:/Windows/System32/OpenSSH/ssh.exe` (9.5).
- **session-bind@openssh.com** (OpenSSH ≥ 8.9): клиент перед подписью привязывает агентную сессию extension-запросом; если агент ответил FAILURE — листинг работает, но подпись отклоняется (`ssh_agent_bind_hostkey: agent refused operation`). Симптом «листинг есть, аутентификации нет» — первым делом проверять это. Наш ответ: перехват в `Extension()` у `WrappedAgent`/`CAPIAgent` (SUCCESS без хранения привязки). Подробности и ссылки — комментарий к константам в `sshagent/server.go`.
- cygwin-транспорт — это TCP на localhost + файл-маркер с портом/UUID (см. `app/cygwin.go`); никакой «несовместимости MSYS AF_UNIX» нет.
- «Too many authentication failures» при многих ключах в агенте: на клиенте лечится `IdentitiesOnly yes` + `IdentityFile <ключ.pub>` (публичного достаточно — подписывает агент), сервер режет после `MaxAuthTries` (по умолчанию 6).

### Отладка агента (проверенная методика)

Тихая смерть/странное поведение GUI-процесса: текстовый лог в `%LOCALAPPDATA%` + heartbeat + `runtime.Stack(all=true)` дампы. Для разбора протокола агента — обёртка `loggingConn` вокруг conn (логировать типы сообщений: 11=list, 13=sign, 27=extension, ответы 6=success, 5=failure, 12=identities). Не предполагать «антивирус», пока не снят дамп стеков.

## Git / push

- Коммиты на русском, формат `[VKCSDEV-XXXXX] ...` — если работа привязана к Jira; в форке допустимо без (история: короткие русские темы).
- Push: `export GIT_SSH="C:/Tools/PuTTY/plink.exe"` + `"C:/Tools/Git/cmd/git.exe" -C <repo> push origin master`. Для работы plink агент должен быть запущен с ключами.
- Перед push делать `fetch`: владелец может аммендить/пушить коммиты параллельно; при non-fast-forward сравнивать деревья (`git diff A B --name-only`), force — только после сравнения.
- Владелец собирает бинарь сам (`build.cmd`); агент собирает только тестовые exe, не коммитит их.
