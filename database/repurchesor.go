package database

import (
	"github.com/twodragon/kore-server/utils"

	"github.com/thoas/go-funk"
)

type RepurchaseList struct {
	Slots []InventorySlot
}

func (r *RepurchaseList) Clear() {
	r.Slots = []InventorySlot{}
}

func (r *RepurchaseList) Push(slot InventorySlot) {
	slot.Activated = false
	slot.InUse = false

	r.Slots = append(r.Slots, slot)
	if len(r.Slots) > 12 {
		r.Slots = r.Slots[len(r.Slots)-12 : len(r.Slots)]
	}
}

func (r *RepurchaseList) Pop(index int) *InventorySlot {

	if len(r.Slots) <= index {
		return nil
	}

	slot := NewSlot()
	*slot = r.Slots[index]

	for i := index; i < len(r.Slots)-1; i++ {
		r.Slots[i] = r.Slots[i+1]
	}

	r.Slots = r.Slots[:len(r.Slots)-1]

	return slot
}

func (r *RepurchaseList) Data(t utils.Packet) []byte {
	resp := utils.Packet{}

	count := byte(len(r.Slots))

	c := 0
	for i := byte(0); i < count; i++ {
		slot := r.Slots[i]
		if slot.ItemID == 0 {
			continue
		}

		r2 := t
		data := slot.GetData(int16(c))
		data = data[6 : len(data)-2]
		r2.Insert(data, 7) // item data

		resp.Concat(r2)
		c++
	}

	return resp
}

func (r *RepurchaseList) HasItem(itemID ...int64) int {
	for i, s := range r.Slots {
		if funk.Contains(itemID, s.ItemID) {
			return i
		}
	}

	return -1
}
