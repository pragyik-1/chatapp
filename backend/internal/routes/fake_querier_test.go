package routes

import (
	"context"
	"errors"

	"chat_app/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// fakeQuerier is a db.Querier whose behaviour each test opts into. Every method
// is implemented, but only the ones a test exercises are given a response; the
// rest fail loudly rather than returning a zero value that could be mistaken for
// a successful query (AGENTS.md 7.1).
type fakeQuerier struct {
	// participantRooms are the rooms IsParticipant answers true for.
	participantRooms map[uuid.UUID]bool
	// isParticipantErr, when set, is returned by IsParticipant.
	isParticipantErr error

	// message, when set, is returned by GetMessageByID.
	message *db.Message
	// getMessageErr, when set, is returned by GetMessageByID.
	getMessageErr error

	// sent, edited, and deleted record that the corresponding write was called.
	sent      []db.SendMessageParams
	edited    []db.EditMessageParams
	deleted   []db.DeleteMessageParams
	sendErr   error
	editErr   error
	deleteErr error
}

func (f *fakeQuerier) IsParticipant(_ context.Context, arg db.IsParticipantParams) (bool, error) {
	if f.isParticipantErr != nil {
		return false, f.isParticipantErr
	}
	return f.participantRooms[arg.RoomID.Bytes], nil
}

func (f *fakeQuerier) GetMessageByID(_ context.Context, id pgtype.UUID) (db.Message, error) {
	if f.getMessageErr != nil {
		return db.Message{}, f.getMessageErr
	}
	if f.message == nil {
		return db.Message{}, pgx.ErrNoRows
	}
	return *f.message, nil
}

func (f *fakeQuerier) SendMessage(_ context.Context, arg db.SendMessageParams) (db.Message, error) {
	if f.sendErr != nil {
		return db.Message{}, f.sendErr
	}
	f.sent = append(f.sent, arg)
	return db.Message{
		ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
		RoomID:    arg.RoomID,
		SenderID:  arg.SenderID,
		Content:   arg.Content,
		ReplyToID: arg.ReplyToID,
	}, nil
}

func (f *fakeQuerier) EditMessage(_ context.Context, arg db.EditMessageParams) (db.Message, error) {
	if f.editErr != nil {
		return db.Message{}, f.editErr
	}
	f.edited = append(f.edited, arg)

	// An edit preserves the room the message already belonged to, so the fake
	// echoes the stored message's room rather than inventing one. A test that
	// does not set message gets a synthetic room, which is enough for tests
	// that only assert on the edit itself.
	roomID := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	if f.message != nil {
		roomID = f.message.RoomID
	}
	return db.Message{
		ID:       arg.ID,
		RoomID:   roomID,
		SenderID: arg.SenderID,
		Content:  arg.Content,
	}, nil
}

func (f *fakeQuerier) DeleteMessage(_ context.Context, arg db.DeleteMessageParams) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, arg)
	return nil
}

// The methods below satisfy the rest of db.Querier. No test in this package
// exercises them, so they fail loudly instead of returning a zero value that a
// caller could mistake for an empty result set (AGENTS.md 7.1).

func (f *fakeQuerier) AddRoomParticipant(context.Context, db.AddRoomParticipantParams) error {
	return errUnusedQuerier
}

func (f *fakeQuerier) CreateRefreshToken(context.Context, db.CreateRefreshTokenParams) (db.RefreshToken, error) {
	return db.RefreshToken{}, errUnusedQuerier
}

func (f *fakeQuerier) CreateRoom(context.Context, db.CreateRoomParams) (db.Room, error) {
	return db.Room{}, errUnusedQuerier
}

func (f *fakeQuerier) CreateUserSettings(context.Context, db.CreateUserSettingsParams) (db.UserSetting, error) {
	return db.UserSetting{}, errUnusedQuerier
}

func (f *fakeQuerier) DeleteExpiredRefreshTokens(context.Context) error {
	return errUnusedQuerier
}

func (f *fakeQuerier) GetRefreshTokenHashByHash(context.Context, string) (db.RefreshToken, error) {
	return db.RefreshToken{}, errUnusedQuerier
}

func (f *fakeQuerier) GetRefreshTokenHashByUserID(context.Context, pgtype.UUID) (db.RefreshToken, error) {
	return db.RefreshToken{}, errUnusedQuerier
}

func (f *fakeQuerier) GetRoomByID(context.Context, pgtype.UUID) (db.Room, error) {
	return db.Room{}, errUnusedQuerier
}

func (f *fakeQuerier) GetRoomMessages(context.Context, db.GetRoomMessagesParams) ([]db.GetRoomMessagesRow, error) {
	return nil, errUnusedQuerier
}

func (f *fakeQuerier) GetRoomParticipants(context.Context, pgtype.UUID) ([]db.GetRoomParticipantsRow, error) {
	return nil, errUnusedQuerier
}

func (f *fakeQuerier) GetUserByEmail(context.Context, string) (db.GetUserByEmailRow, error) {
	return db.GetUserByEmailRow{}, errUnusedQuerier
}

func (f *fakeQuerier) GetUserByID(context.Context, pgtype.UUID) (db.GetUserByIDRow, error) {
	return db.GetUserByIDRow{}, errUnusedQuerier
}

func (f *fakeQuerier) GetUserRooms(context.Context, pgtype.UUID) ([]db.Room, error) {
	return nil, errUnusedQuerier
}

func (f *fakeQuerier) GetUserSettings(context.Context, pgtype.UUID) (db.UserSetting, error) {
	return db.UserSetting{}, errUnusedQuerier
}

func (f *fakeQuerier) RegisterUser(context.Context, db.RegisterUserParams) (db.RegisterUserRow, error) {
	return db.RegisterUserRow{}, errUnusedQuerier
}

func (f *fakeQuerier) RegisterUserWithSettings(context.Context, db.RegisterUserWithSettingsParams) (db.RegisterUserWithSettingsRow, error) {
	return db.RegisterUserWithSettingsRow{}, errUnusedQuerier
}

func (f *fakeQuerier) RemoveRoomParticipant(context.Context, db.RemoveRoomParticipantParams) error {
	return errUnusedQuerier
}

func (f *fakeQuerier) RevokeRefreshToken(context.Context, string) error {
	return errUnusedQuerier
}

func (f *fakeQuerier) RevokeRefreshTokensByUserID(context.Context, pgtype.UUID) error {
	return errUnusedQuerier
}

func (f *fakeQuerier) SearchUsers(context.Context, db.SearchUsersParams) ([]db.SearchUsersRow, error) {
	return nil, errUnusedQuerier
}

func (f *fakeQuerier) UpdateUserSettings(context.Context, db.UpdateUserSettingsParams) (db.UserSetting, error) {
	return db.UserSetting{}, errUnusedQuerier
}

// errUnusedQuerier marks a db.Querier method a test did not expect to reach.
var errUnusedQuerier = errors.New("fakeQuerier: method not expected in this test")
