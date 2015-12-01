package topom

import (
	"testing"

	"github.com/wandoulabs/codis/pkg/models"
	"github.com/wandoulabs/codis/pkg/utils/assert"
)

func TestSlotState(x *testing.T) {
	t := openTopom()
	defer t.Close()

	const sid = 1
	const gid1 = 1
	const gid2 = 2
	const server1 = "server1"
	const server2 = "server2"

	check := func() {
		ctx, err := t.newContext()
		assert.MustNoError(err)
		m, err := ctx.getSlotMapping(sid)
		assert.MustNoError(err)
		assert.Must(m.Id == sid)
		assert.Must(m.GroupId == gid1 && m.Action.TargetId == gid2)

		g1, err := ctx.getGroup(gid1)
		assert.MustNoError(err)
		assert.Must(ctx.getGroupMaster(gid1) == server1)

		g2, err := ctx.getGroup(gid2)
		assert.MustNoError(err)
		assert.Must(ctx.getGroupMaster(gid2) == server2)

		slot := ctx.toSlot(m)

		switch m.Action.State {
		case models.ActionPrepared:
			assert.Must(slot.Locked == true)
		case models.ActionMigrating:
			switch {
			case g1.Promoting.State == models.ActionPrepared:
				assert.Must(slot.Locked == true)
			case g2.Promoting.State == models.ActionPrepared:
				assert.Must(slot.Locked == true)
			default:
				assert.Must(slot.Locked == false)
				assert.Must(slot.BackendAddr == server2)
				assert.Must(slot.MigrateFrom == server1)
			}
		case models.ActionFinished:
			switch {
			case g2.Promoting.State == models.ActionPrepared:
				assert.Must(slot.Locked == true)
			default:
				assert.Must(slot.Locked == false)
				assert.Must(slot.BackendAddr == server2)
				assert.Must(slot.MigrateFrom == "")
			}
		default:
			switch {
			case g1.Promoting.State == models.ActionPrepared:
				assert.Must(slot.Locked == true)
			default:
				assert.Must(slot.Locked == false)
				assert.Must(slot.BackendAddr == server1)
				assert.Must(slot.MigrateFrom == "")
			}
		}
	}

	g1 := &models.Group{Id: gid1}
	g1.Servers = append(g1.Servers, &models.GroupServer{Addr: server1})
	g2 := &models.Group{Id: gid2}
	g2.Servers = append(g2.Servers, &models.GroupServer{Addr: server2})

	m := &models.SlotMapping{Id: sid}
	m.GroupId = gid1
	m.Action.TargetId = gid2

	sstates := []string{
		models.ActionNothing,
		models.ActionPending,
		models.ActionPreparing,
		models.ActionPrepared,
		models.ActionMigrating,
		models.ActionFinished,
	}

	gstates := []string{
		models.ActionNothing,
		models.ActionPreparing,
		models.ActionPrepared,
		models.ActionFinished,
	}

	for _, m.Action.State = range sstates {
		t.dirtySlotsCache(m.Id)
		assert.MustNoError(t.storeUpdateSlotMapping(m))
		for _, g1.Promoting.State = range gstates {
			t.dirtyGroupCache(g1.Id)
			assert.MustNoError(t.storeUpdateGroup(g1))
			for _, g2.Promoting.State = range gstates {
				t.dirtyGroupCache(g2.Id)
				assert.MustNoError(t.storeUpdateGroup(g2))
				check()
			}
		}
	}
}
