package storage

type CurrentGameSession struct {
	SessionID int64
	Game      string
}

// ActiveGameSessions хранит активные игровые сессии в памяти
// key = Discord userID
var ActiveGameSessions = make(map[string]CurrentGameSession)
