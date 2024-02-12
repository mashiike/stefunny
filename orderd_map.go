package stefunny

type OrderdMap[K comparable, V any] struct {
	keys   []K
	values map[K]V
}

func NewOrderdMap[K comparable, V any]() *OrderdMap[K, V] {
	return &OrderdMap[K, V]{
		keys:   make([]K, 0),
		values: make(map[K]V),
	}
}

func (m *OrderdMap[K, V]) Set(key K, value V) {
	if _, ok := m.values[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

func (m *OrderdMap[K, V]) Get(key K) (V, bool) {
	v, ok := m.values[key]
	return v, ok
}

func (m *OrderdMap[K, V]) Keys() []K {
	return m.keys
}

func (m *OrderdMap[K, V]) Values() []V {
	result := make([]V, 0, len(m.keys))
	for _, k := range m.keys {
		result = append(result, m.values[k])
	}
	return result
}
