package recognizer

// JumpTable задает функцию переходов лексического распознавателя в виде таблицы
type JumpTable struct {
	table         [][]int
	factorization map[byte]int
}

// NewJumpTable создает JumpTable
func NewJumpTable(table [][]int, factorization map[byte]int) JumpTable {
	return JumpTable{
		table:         table,
		factorization: factorization,
	}
}

// Next возвращает состояние, в которое осуществляется
// переход из state по символу symbol
func (t *JumpTable) Next(state int, symbol byte) int {
	return t.table[state][t.factorization[symbol]]
}
