package utco

// Store 是 UTCO 集的内存工作态容器，承载当前已确认的未转移凭信输出。
// 它是状态层的领域对象（供解析、状态转移、到期清理、指纹计算与快照使用），
// 不是持久化存储——长期落盘与网络同步不属于本层职责。
type Store struct {
	// entries 以完整 OutPoint 为键保存 entry。已转出的 entry 置 Spent=true；
	// 过期清理在区块结算时进行（见 expiry.go）。
	entries map[OutPoint]Entry
}

// NewStore 构造一个空的内存 UTCO 集。
func NewStore() *Store {
	return &Store{entries: make(map[OutPoint]Entry)}
}

// Put 插入一条 UTCO 记录。若该 OutPoint 已存在则返回 ErrDuplicate。
func (s *Store) Put(e Entry) error {
	op := e.OutPoint()
	if _, ok := s.entries[op]; ok {
		return ErrDuplicate
	}
	s.entries[op] = e
	return nil
}

// Get 按完整 OutPoint 查询 entry。未命中返回 ErrNotFound。
func (s *Store) Get(op OutPoint) (Entry, error) {
	e, ok := s.entries[op]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return e, nil
}

// Spend 将指定凭信标记为已转出/消费（一次性转移）。未命中返回 ErrNotFound；
// 已转出再次转移返回 ErrAlreadySpent。
func (s *Store) Spend(op OutPoint) error {
	e, ok := s.entries[op]
	if !ok {
		return ErrNotFound
	}
	if e.Spent {
		return ErrAlreadySpent
	}
	e.Spent = true
	s.entries[op] = e
	return nil
}

// ValidEntries 返回所有未转出（有效）的 entry 副本，供局部引用解析与状态
// 指纹计算使用。注意：过期但尚未清理的凭信仍可能在此返回，过期过滤由调用方
// 结合 Entry.Expired 或区块结算时的 expiry 流程处理。返回顺序不保证稳定。
func (s *Store) ValidEntries() []Entry {
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		if !e.Spent {
			out = append(out, e)
		}
	}
	return out
}
