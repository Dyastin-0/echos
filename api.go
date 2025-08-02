package echos

import (
	"encoding/json"
	"net/http"
	"slices"
)

type HTTPresponse map[string]string

func (e *Echos) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowed := slices.Contains(allowedOrigins, origin)

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (e *Echos) CreateRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := GenerateMeetRoomID(3, 3)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	// TODO: hande room id collisions
	if _, ok := e.Rooms.Load(roomID); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(HTTPresponse{
			"error": "failed to create a room, try again",
		})
		return
	}

	e.Rooms.Store(roomID, NewRoom(roomID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(HTTPresponse{
		"message": "room created successfully",
		"room":    roomID,
	})
}

func (e *Echos) CheckRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")

	if roomID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HTTPresponse{
			"error": "missing room id",
		})
		return
	}

	_, exists := e.Rooms.Load(roomID)

	if exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HTTPresponse{
			"message": "room exists",
			"room":    roomID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(HTTPresponse{
		"error": "room not found",
	})
}
