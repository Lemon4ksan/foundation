package list

const chunkSize = 256

type listChunk[T any] [chunkSize]Element[T]

func (l *List[T]) elem(idx int) *Element[T] {
	return &l.chunks[idx/chunkSize][idx%chunkSize]
}
