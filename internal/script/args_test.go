package script

import "testing"

// TestArgsAreaFIFO 测试实参区 FIFO 顺序。
func TestArgsAreaFIFO(t *testing.T) {
	a := NewArgsArea()
	a.Enqueue(IntValue(1))
	a.Enqueue(IntValue(2))
	a.Enqueue(IntValue(3))

	for want := int64(1); want <= 3; want++ {
		v, err := a.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue error: %v", err)
		}
		n, _ := v.AsInt()
		if n != want {
			t.Errorf("Dequeue = %d, want %d", n, want)
		}
	}
	if a.Len() != 0 {
		t.Errorf("Len after full dequeue = %d, want 0", a.Len())
	}
}

// TestArgsAreaEmptyDequeue 测试空实参区取值返回 ErrArgCountMismatch。
func TestArgsAreaEmptyDequeue(t *testing.T) {
	a := NewArgsArea()
	if _, err := a.Dequeue(); err != ErrArgCountMismatch {
		t.Errorf("Dequeue on empty: got %v, want ErrArgCountMismatch", err)
	}
}

// TestArgsAreaClear 测试 Clear 后实参区为空。
func TestArgsAreaClear(t *testing.T) {
	a := NewArgsArea()
	a.Enqueue(BoolValue(true))
	a.Enqueue(BoolValue(false))
	a.Clear()
	if a.Len() != 0 {
		t.Errorf("Len after Clear = %d, want 0", a.Len())
	}
	if _, err := a.Dequeue(); err != ErrArgCountMismatch {
		t.Error("Dequeue on cleared args should return ErrArgCountMismatch")
	}
}

// TestArgsAreaItems 测试 Items 返回正确副本。
func TestArgsAreaItems(t *testing.T) {
	a := NewArgsArea()
	a.Enqueue(IntValue(10))
	a.Enqueue(IntValue(20))
	items := a.Items()
	if len(items) != 2 {
		t.Fatalf("Items len = %d, want 2", len(items))
	}
	n0, _ := items[0].AsInt()
	n1, _ := items[1].AsInt()
	if n0 != 10 || n1 != 20 {
		t.Errorf("Items = [%d, %d], want [10, 20]", n0, n1)
	}
	// Items 返回副本，不影响实参区
	if a.Len() != 2 {
		t.Error("Items should not consume args")
	}
}
