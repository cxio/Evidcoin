package script

// ArgsArea 是脚本 VM 实参区（FIFO）。
// 前置截取指令（AT/@、SAVE/$、DIRECT/~）将值压入实参区；
// 指令执行时按先进先出顺序取出实参。
// 参考：docs/proposal/10.Script-System.md §2，DEC-0501。
type ArgsArea struct {
	items []Value
}

// NewArgsArea 创建一个空实参区。
func NewArgsArea() *ArgsArea { return &ArgsArea{} }

// Enqueue 向实参区尾部追加一个值。
func (a *ArgsArea) Enqueue(v Value) {
	a.items = append(a.items, v)
}

// Dequeue 从实参区头部取出一个值（FIFO）。
// 实参区为空时返回 ErrArgCountMismatch。
func (a *ArgsArea) Dequeue() (Value, error) {
	if len(a.items) == 0 {
		return Value{}, ErrArgCountMismatch
	}
	v := a.items[0]
	a.items = a.items[1:]
	return v, nil
}

// Len 返回实参区中当前值数量。
func (a *ArgsArea) Len() int { return len(a.items) }

// Clear 清空实参区（指令执行后调用，防止残留）。
func (a *ArgsArea) Clear() { a.items = a.items[:0] }

// Items 返回实参区所有值的副本（不修改实参区，从队头到队尾顺序）。
func (a *ArgsArea) Items() []Value {
	cp := make([]Value, len(a.items))
	copy(cp, a.items)
	return cp
}
