package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestFriends_SendRequest_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	sender, token := createUser(t, pool)
	target, _ := createUser(t, pool)
	_ = sender

	w := req(router, http.MethodPost, "/api/v1/friends/request",
		map[string]any{"addressee_id": target.ID.String()},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := jsonBody(t, w)
	friendshipID := mustString(t, body, "friendship_id")
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM friendships WHERE id = $1`, friendshipID)
	})
}

func TestFriends_SendRequest_ToSelf_Returns422(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	user, token := createUser(t, pool)

	w := req(router, http.MethodPost, "/api/v1/friends/request",
		map[string]any{"addressee_id": user.ID.String()},
		"Authorization", bearer(token),
	)

	// ErrCannotFriendSelf maps to 400 in the handler
	if w.Code != http.StatusBadRequest && w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 400/422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_SendRequest_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodPost, "/api/v1/friends/request",
		map[string]any{"addressee_id": uuid.New().String()},
	)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_SendRequest_Duplicate_Returns409(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	sender, token := createUser(t, pool)
	target, _ := createUser(t, pool)

	_, err := pool.Exec(context.Background(),
		`INSERT INTO friendships (requester_id, addressee_id, status) VALUES ($1, $2, 'pending')`,
		sender.ID, target.ID,
	)
	if err != nil {
		t.Fatalf("seed friendship: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(),
			`DELETE FROM friendships WHERE requester_id = $1 AND addressee_id = $2`,
			sender.ID, target.ID,
		)
	})

	w := req(router, http.MethodPost, "/api/v1/friends/request",
		map[string]any{"addressee_id": target.ID.String()},
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_AcceptRequest_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	sender, _ := createUser(t, pool)
	addressee, addresseeToken := createUser(t, pool)

	var friendshipID uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO friendships (requester_id, addressee_id, status) VALUES ($1, $2, 'pending') RETURNING id`,
		sender.ID, addressee.ID,
	).Scan(&friendshipID)
	if err != nil {
		t.Fatalf("seed friendship: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM friendships WHERE id = $1`, friendshipID)
	})

	w := req(router, http.MethodPost, "/api/v1/friends/accept/"+friendshipID.String(), nil,
		"Authorization", bearer(addresseeToken),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_AcceptRequest_NotAddressee_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	sender, senderToken := createUser(t, pool)
	addressee, _ := createUser(t, pool)

	var friendshipID uuid.UUID
	pool.QueryRow(context.Background(),
		`INSERT INTO friendships (requester_id, addressee_id, status) VALUES ($1, $2, 'pending') RETURNING id`,
		sender.ID, addressee.ID,
	).Scan(&friendshipID)
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM friendships WHERE id = $1`, friendshipID)
	})

	// Sender tries to accept their own request — only addressee may accept
	w := req(router, http.MethodPost, "/api/v1/friends/accept/"+friendshipID.String(), nil,
		"Authorization", bearer(senderToken),
	)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_RejectRequest_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	sender, _ := createUser(t, pool)
	addressee, addresseeToken := createUser(t, pool)

	var friendshipID uuid.UUID
	pool.QueryRow(context.Background(),
		`INSERT INTO friendships (requester_id, addressee_id, status) VALUES ($1, $2, 'pending') RETURNING id`,
		sender.ID, addressee.ID,
	).Scan(&friendshipID)

	w := req(router, http.MethodDelete, "/api/v1/friends/reject/"+friendshipID.String(), nil,
		"Authorization", bearer(addresseeToken),
	)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_Unfriend_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	userA, tokenA := createUser(t, pool)
	userB, _ := createUser(t, pool)

	var friendshipID uuid.UUID
	pool.QueryRow(context.Background(),
		`INSERT INTO friendships (requester_id, addressee_id, status) VALUES ($1, $2, 'accepted') RETURNING id`,
		userA.ID, userB.ID,
	).Scan(&friendshipID)

	w := req(router, http.MethodDelete, "/api/v1/friends/"+friendshipID.String(), nil,
		"Authorization", bearer(tokenA),
	)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_Unfriend_NotParticipant_Returns403(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	userA, _ := createUser(t, pool)
	userB, _ := createUser(t, pool)
	outsider, outsiderToken := createUser(t, pool)
	_ = outsider

	var friendshipID uuid.UUID
	pool.QueryRow(context.Background(),
		`INSERT INTO friendships (requester_id, addressee_id, status) VALUES ($1, $2, 'accepted') RETURNING id`,
		userA.ID, userB.ID,
	).Scan(&friendshipID)
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM friendships WHERE id = $1`, friendshipID)
	})

	w := req(router, http.MethodDelete, "/api/v1/friends/"+friendshipID.String(), nil,
		"Authorization", bearer(outsiderToken),
	)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_ListFriends_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/friends", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_ListFriends_NoAuth_Returns401(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)

	w := req(router, http.MethodGet, "/api/v1/friends", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_ListRequests_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/friends/requests", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFriends_ListOutgoing_Success(t *testing.T) {
	pool := newPool(t)
	router := newRouter(pool)
	_, token := createUser(t, pool)

	w := req(router, http.MethodGet, "/api/v1/friends/outgoing", nil,
		"Authorization", bearer(token),
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
