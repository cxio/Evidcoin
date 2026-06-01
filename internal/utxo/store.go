package utxo

// Store 是 UTXO 集的内存工作态容器，承载当前已确认的未花费币金输出。
// 它是状态层的领域对象（供解析、状态转移、指纹计算与快照使用），不是持久化
// 存储——长期落盘、网络同步与 Blockqs 回填不属于本层职责（见仓库分层约定）。
type Store struct {
	// entries 以完整 OutPoint 为键保存 entry。已花费的 entry 仍保留并置
	// Spent=true，直到区块结算时按指纹规则决定移除（Count 减至零删叶）。
	entries map[OutPoint]Entry
}

// NewStore 构造一个空的内存 UTXO 集。
func NewStore() *Store {
	return &Store{entries: make(map[OutPoint]Entry)}
}

// Put 插入一条 UTXO 记录。若该 OutPoint 已存在则返回 ErrDuplicate，
// 以防止重复确认同一输出。
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

// Spend 将指定 OutPoint 标记为已花费。未命中返回 ErrNotFound；
// 已花费再次花费返回 ErrAlreadySpent（同批次/历史重复消费保护）。
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

// ValidEntries 返回所有未花费（有效）的 entry 副本，供局部引用解析与状态
// 指纹计算使用。返回顺序不保证稳定，调用方需自行按协议规则排序。
func (s *Store) ValidEntries() []Entry {
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		if !e.Spent {
			out = append(out, e)
		}
	}
	return out
}
