package back_test

import (
	"kaepora/internal/back"
	"kaepora/internal/util"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMatchSessionUUIDs(t *testing.T) {
	sess := back.NewMatchSession(util.NewUUIDAsBlob(), time.Now())

	u1, u2, u3, u4 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	u1[0], u2[0], u3[0], u4[0] = 1, 2, 3, 4

	// Ensure order is conserved.
	sess.AddPlayerID(u1, u2, u3, u1, u4, u3)
	sess.RemovePlayerID(u1)
	expected := []uuid.UUID{u2, u3, u4}
	actual := sess.GetPlayerIDs()

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected %v\ngot %v", expected, actual)
	}
}
