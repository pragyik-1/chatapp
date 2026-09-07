package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateRoomRequest struct {
	Name      string    `json:"name"`
	IsGroup   bool      `json:"is_group"`
	CreatedBy uuid.UUID `json:"created_by"`
}

func createRoom(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateRoomRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.Name == "" && req.IsGroup {
			utils.WriteError(w, http.StatusBadRequest, "room name is required")
			return
		}

		if req.CreatedBy == (uuid.UUID{}) {
			utils.WriteError(w, http.StatusBadRequest, "created by is required")
			return
		}

		userId := pgtype.UUID{Bytes: req.CreatedBy, Valid: true}

		if _, err := queries.GetUserByID(r.Context(), userId); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				utils.WriteError(w, http.StatusNotFound, "created_by user does not exist")
				return
			}
			utils.WriteError(w, http.StatusInternalServerError, "failed to create room")
			return
		}

		room, err := queries.CreateRoom(r.Context(), db.CreateRoomParams{
			Name:      pgtype.Text{String: req.Name, Valid: req.Name != ""},
			IsGroup:   pgtype.Bool{Bool: req.IsGroup, Valid: true},
			CreatedBy: userId,
		})

		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to create room")
			return
		}

		err = queries.AddRoomParticipant(r.Context(), db.AddRoomParticipantParams{
			RoomID: room.ID,
			UserID: room.CreatedBy,
		})

		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to add participant")
			return
		}

		utils.WriteJSON(w, http.StatusCreated, room)
	}
}
