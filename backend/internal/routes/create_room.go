package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateRoomRequest struct {
	Name    string `json:"name"`
	IsGroup bool   `json:"is_group"`
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

		userID, ok := utils.GetUserIDFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		userId := pgtype.UUID{Bytes: userID, Valid: true}

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
