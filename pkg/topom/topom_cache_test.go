package topom

import (
	"path/filepath"
	"testing"

	"github.com/wandoulabs/codis/pkg/models"
	"github.com/wandoulabs/codis/pkg/utils/assert"
	"github.com/wandoulabs/codis/pkg/utils/errors"
)

func TestSlotsCache(x *testing.T) {
	t := openTopom()
	defer t.Close()

	const sid = 100

	check := func(gid int) {
		ctx, err := t.newContext()
		assert.MustNoError(err)
		m, err := ctx.getSlotMapping(sid)
		assert.MustNoError(err)
		assert.Must(m.Id == sid && m.GroupId == gid)
	}

	m := &models.SlotMapping{Id: sid}
	check(0)

	t.dirtySlotsCache(sid)
	m.GroupId = 100
	assert.MustNoError(t.storeUpdateSlotMapping(m))
	check(100)

	t.dirtySlotsCache(sid)
	m.GroupId = 200
	check(100)

	t.dirtyCacheAll()
	m.GroupId = 200
	check(100)

	t.dirtyCacheAll()
	m.GroupId = 300
	assert.MustNoError(t.storeUpdateSlotMapping(m))
	check(300)
}

func TestGroupCache(x *testing.T) {
	t := openTopom()
	defer t.Close()

	const gid = 100

	check := func(exists bool, state string) {
		ctx, err := t.newContext()
		assert.MustNoError(err)
		if !exists {
			assert.Must(ctx.group[gid] == nil)
		} else {
			g, err := ctx.getGroup(gid)
			assert.MustNoError(err)
			assert.Must(g.Id == gid && g.Promoting.State == state)
		}
	}

	g := &models.Group{Id: gid}
	check(false, "")

	t.dirtyGroupCache(gid)
	check(false, "")

	t.dirtyGroupCache(gid)
	assert.MustNoError(t.storeCreateGroup(g))
	check(true, models.ActionNothing)

	t.dirtyGroupCache(gid)
	g.Promoting.State = models.ActionPreparing
	check(true, models.ActionNothing)

	t.dirtyGroupCache(gid)
	g.Promoting.State = models.ActionPreparing
	assert.MustNoError(t.storeUpdateGroup(g))
	check(true, models.ActionPreparing)

	t.dirtyCacheAll()
	g.Promoting.State = models.ActionPrepared
	assert.MustNoError(t.storeUpdateGroup(g))
	check(true, models.ActionPrepared)

	t.dirtyGroupCache(gid)
	assert.MustNoError(t.storeRemoveGroup(g))
	check(false, "")
}

func TestProxyCache(x *testing.T) {
	t := openTopom()
	defer t.Close()

	const token = "fake_proxy_token"

	check := func(exists bool) {
		ctx, err := t.newContext()
		assert.MustNoError(err)
		if !exists {
			assert.Must(ctx.proxy[token] == nil)
		} else {
			p, err := ctx.getProxy(token)
			assert.MustNoError(err)
			assert.Must(p.Token == token)
		}
	}

	p := &models.Proxy{Token: token}
	check(false)

	t.dirtyProxyCache(p.Token)
	assert.MustNoError(t.storeCreateProxy(p))
	check(true)

	t.dirtyProxyCache(p.Token)
	assert.MustNoError(t.storeRemoveProxy(p))
	check(false)
}

type memStore struct {
	data map[string][]byte
}

func newMemStore() *memStore {
	return &memStore{make(map[string][]byte)}
}

type memClient struct {
	*memStore
}

func newMemClient(store *memStore) models.Client {
	if store == nil {
		store = newMemStore()
	}
	return &memClient{store}
}

func (c *memClient) Create(path string, data []byte) error {
	if _, ok := c.data[path]; ok {
		return errors.Errorf("node already exists")
	}
	c.data[path] = data
	return nil
}

func (c *memClient) Update(path string, data []byte) error {
	c.data[path] = data
	return nil
}

func (c *memClient) Delete(path string) error {
	delete(c.data, path)
	return nil
}

func (c *memClient) Read(path string) ([]byte, error) {
	return c.data[path], nil
}

func (c *memClient) List(path string) ([]string, error) {
	path = filepath.Clean(path)
	var list []string
	for k, _ := range c.data {
		if path == filepath.Dir(k) {
			list = append(list, k)
		}
	}
	return list, nil
}

func (c *memClient) Close() error {
	return nil
}
