package echos

import (
	"encoding/json"
	"net/http"
)

type HTTPresponse map[string]string

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

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

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")

	if room == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HTTPresponse{
			"error": "Room name is required",
		})
		return
	}

	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	if _, ok := Rooms[room]; ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(HTTPresponse{
			"error": "Room already exists",
		})
		return
	}

	Rooms[room] = NewRoom(room)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(HTTPresponse{
		"message": "Room created successfully",
		"room":    room,
	})
}

func CheckRoom(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")

	if room == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HTTPresponse{
			"error": "Room name is required",
		})
		return
	}

	roomsMutex.RLock()
	_, exists := Rooms[room]
	roomsMutex.RUnlock()

	if exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HTTPresponse{
			"message": "Room exists",
			"room":    room,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(HTTPresponse{
		"error": "Room not found",
	})
}
