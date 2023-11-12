package player

import (
	"time"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/utils"
)

type (
	SendOmokRequestHandler    struct{}
	RespondOmokRequestHandler struct{}
	AddOmokPointHandler       struct{}
	CloseOmokHandler          struct{}

	OmokGame struct {
		Player1 *database.Character
		Player2 *database.Character
		Turn    int
		Board   [19][19]int
	}
)

var (
	OmokGames = make(map[int]*OmokGame)

	SEND_OMOK_REQUEST = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xa4, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	OMOK_ACCEPTED     = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0xa4, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	ADD_POINT         = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0xa4, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	OMOK_WIN          = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0xa4, 0x04, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

func (h *SendOmokRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	if opponent == nil {
		return nil, nil
	}
	if opponent.OmokID != 0 || opponent.OmokRequestState != 0 || s.Character.OmokID != 0 || s.Character.OmokRequestState != 0 {
		return messaging.SystemMessage(41014), nil //already playing omok
	}
	resp := SEND_OMOK_REQUEST
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 8)
	opponent.Socket.Write(resp)
	s.Character.OmokRequestState = 1
	opponent.OmokRequestState = 1

	sec := 10
	for sec >= 0 {
		time.Sleep(time.Second * 1)
		sec--
		if sec <= 0 {
			opponent.OmokRequestState = 0
			s.Character.OmokRequestState = 0
			opponent.Socket.Write(messaging.SystemMessage(41006))
			return messaging.SystemMessage(41006), nil
		}
		if opponent.OmokRequestState == 2 {
			s.Character.OmokRequestState = 0
			opponent.OmokRequestState = 0
			return nil, nil
		}
		if opponent.OmokRequestState == 0 {
			opponent.Socket.Write(messaging.SystemMessage(41006))
			opponent.OmokRequestState = 0
			s.Character.OmokRequestState = 0
			return messaging.SystemMessage(41006), nil
		}

	}
	return nil, nil
}
func (h *RespondOmokRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[7:9], true))
	accepted := data[6] == 1

	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	if opponent == nil {
		return nil, nil
	}

	if !accepted {
		opponent.OmokRequestState = 0
		s.Character.OmokRequestState = 0
		resp := messaging.SystemMessage(messaging.OMOK_REQUEST_REJECTED)
		opponent.Socket.Write(resp)
		return resp, nil

	} else { // start pvp
		opponent.OmokRequestState = 2
		s.Character.OmokRequestState = 2

		//create new omok game and add it to the map
		game := &OmokGame{
			Player1: s.Character,
			Player2: opponent,
			Turn:    1,
			Board:   [19][19]int{},
		}
		OmokGames[s.Character.ID] = game

		s.Character.OmokID = s.Character.ID
		opponent.OmokID = s.Character.ID
		s.Character.Opponent = pseudoID
		opponent.Opponent = s.Character.PseudoID

		resp := OMOK_ACCEPTED
		opponent.Socket.Write(resp)

	}
	return nil, nil
}
func (h *AddOmokPointHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	x := int(utils.BytesToInt(data[6:8], true))
	y := int(utils.BytesToInt(data[8:10], true))

	resp := ADD_POINT
	resp.Insert(utils.IntToBytes(uint64(x), 2, true), 8)
	resp.Insert(utils.IntToBytes(uint64(y), 2, true), 10)

	//add point to the board and check if player has won

	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, s.Character.Opponent)
	if opponent == nil {
		return nil, nil
	}
	game, ok := OmokGames[s.Character.OmokID]
	if !ok {
		return nil, nil
	}
	player := AddPoint(game, x, y, game.Turn)
	if player == 1 || player == 2 || player == 3 {
		resp = OMOK_WIN
		resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 10)
		s.Write(resp)
		sock := database.GetSocket(opponent.UserID)
		sock.Write(resp)
		game.EndOmok()

		return nil, nil
	}

	sock := database.GetSocket(opponent.UserID)
	sock.Write(resp)
	return nil, nil
}
func (h *CloseOmokHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, s.Character.Opponent)
	if opponent == nil {
		return nil, nil
	}

	resp := OMOK_WIN
	resp.Insert(utils.IntToBytes(uint64(opponent.PseudoID), 2, true), 10)
	if opponent.Socket != nil {
		opponent.Socket.Write(resp)
	}
	OmokGames[s.Character.OmokID].EndOmok()

	return resp, nil
}

// the game omok (GO) is a two player game where each player has a board of size 19x19.
// The board is divided into 19x19 squares.
// Each square is a point.
// The first player to get 5 points in a row (horizontally, vertically or diagonally) wins.
// The game ends when one of the players has 5 points in a row.
// Function must pass the following parameters: pointer to OmokGame x, y, player
// x and y are the coordinates of the point.
// player is the player who placed the point.
// The function must return the following values:
// winner: the player who won the game.
// -1: the game is not finished.
// 0: the game is finished and no one won.
// 1: the game is finished and player 1 won.
// 2: the game is finished and player 2 won.

func AddPoint(game *OmokGame, x, y int, player int) int {
	if game.Board[x][y] != 0 {
		return -1
	}
	if player == 1 {
		game.Turn = 2
	}
	if player == 2 {
		game.Turn = 1
	}
	game.Board[x][y] = player

	// check horizontal if player has 5 points in a row
	points := 0
	for i := 0; i < 19; i++ {
		if game.Board[x][i] == player {
			points++
		} else {
			points = 0
		}
		if points == 5 {
			return player
		}
	}

	// check vertical if player has 5 points in a row
	points = 0
	for i := 0; i < 19; i++ {
		if game.Board[i][y] == player {
			points++
		} else {
			points = 0
		}
		if points == 5 {
			return player
		}
	}

	//check every diagonal if player has 5 points in a row
	points = 0
	for i := 0; i < 19; i++ {
		if game.Board[i][i] == player {
			points++
		} else {
			points = 0
		}
		if points == 5 {
			return player
		}
	}

	//check every opposite diagonal if player has 5 points in a row
	points = 0
	for i := 0; i < 19; i++ {
		if game.Board[i][18-i] == player {
			points++
		} else {
			points = 0
		}
		if points == 5 {
			return player
		}
	}

	return -1
}

func (h *OmokGame) EndOmok() {
	h.Player1.OmokID = 0
	h.Player1.Opponent = 0
	h.Player2.OmokID = 0
	h.Player2.Opponent = 0
	delete(OmokGames, h.Player1.ID)
	delete(OmokGames, h.Player2.ID)
}
