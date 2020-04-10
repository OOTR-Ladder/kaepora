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

	u1 := uuid.New()
	u2 := uuid.New()
	u3 := uuid.New()
	u4 := uuid.New()

	sess.AddPlayerID(u1, u2, u3, u4)
	expected := []uuid.UUID{u1, u2, u3, u4}
	actual := sess.GetPlayerIDs()

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected %v\ngot %v", expected, actual)
	}
}
