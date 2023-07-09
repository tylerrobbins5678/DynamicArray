package mappedlist

import "sync"

// Prefix fields are strictly seperate from others.
// ie. prefix length not included in length
type Mappedlist[T any] struct {
	Data            []*[arraySize]T
	MapSize         int
	AllocSize       int
	Length          int
	PrefixMapSize   int
	PrefixAllocSize int
	PrefixLength    int
}

// size of newly instantiated array. range 1 - 16777216,
// smaller -> more overhead cost / high memory efficiency,
// larger -> more memory consumption / much faster,
// 1024 is suggested for average use case,
// 32768+ is suggested for analsys operations
const arraySize = 1024

// devisor of added growth ie. 4 -> size = size + (size / 4)
const growthFactor = 4

// create mappedarray
func Make[T any]() Mappedlist[T] {
	// instantiate data struct
	data := make([]*[arraySize]T, 1)
	data[0] = GenerateInner[T]()
	res := Mappedlist[T]{Data: data,
		MapSize:         1,
		AllocSize:       arraySize,
		Length:          0,
		PrefixMapSize:   0,
		PrefixAllocSize: 0,
		PrefixLength:    0,
	}

	return res
}

// multithreaded create from slice, much faster than calling append
func MakeFromArray[T any](arr []T) Mappedlist[T] {
	data := make([]*[arraySize]T, len(arr)/arraySize+1)
	var wg sync.WaitGroup
	// i is offset
	for i := 0; i < len(arr)/arraySize+1; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			innerData := GenerateInner[T]()
			for j := 0; j < arraySize; j++ {
				if i*arraySize+j >= len(arr) {
					break
				} else {
					innerData[j] = arr[i*arraySize+j]
				}
			}
			data[i] = innerData
		}(i)
	}
	wg.Wait()

	res := Mappedlist[T]{
		Data:            data,
		MapSize:         len(arr)/arraySize + 1,
		AllocSize:       ((len(arr) / arraySize) + 1) * arraySize,
		Length:          len(arr),
		PrefixMapSize:   0,
		PrefixAllocSize: 0,
		PrefixLength:    0,
	}
	return res
}

// returns slice
// TODO multithread operation similar to makeFromArray func
func (list *Mappedlist[T]) ToArray() []T {

	res := make([]T, list.PrefixLength+list.Length)

	for i := 0; i < (list.PrefixLength + list.Length); i++ {
		res[i] = list.Get(i)
	}

	return res
}

// returns empty instanciated array pointer
func GenerateInner[T any]() *[arraySize]T {
	var res [arraySize]T
	return &res
}

// get element at index
func (list *Mappedlist[T]) Get(index int) T {
	index += list.PrefixMapSize*arraySize - list.PrefixLength
	return list.Data[index/arraySize][index%arraySize]
}

// set element at index
func (list *Mappedlist[T]) set(index int, element T) {
	index += list.PrefixMapSize*arraySize - list.PrefixLength
	list.Data[index/arraySize][index%arraySize] = element
}

// returns next size that map should be
func getNextSize(size int) int {
	// allocate size / growthFactor more if over 1024, 100% more if less than 1024
	if size > 1024 {
		return size / growthFactor
	} else {
		return size
	}
}

// allocate space at end of mappedlist
func (list *Mappedlist[T]) allocNextInner() {
	// if out of alloc space
	if list.AllocSize == list.MapSize*arraySize {
		// make new data and copy to new
		toAdd := getNextSize(list.MapSize)
		newSize := toAdd + list.PrefixMapSize + list.MapSize
		newData := make([]*[arraySize]T, newSize)

		for i := 0; i < list.MapSize+list.PrefixMapSize; i++ {
			newData[i] = list.Data[i]
		}
		list.Data = newData
		list.MapSize += toAdd
	}

	// alloc next sector
	list.Data[list.PrefixMapSize+list.AllocSize/arraySize] = GenerateInner[T]()
	list.AllocSize += arraySize
}

// only allocate single array
func (list *Mappedlist[T]) allocPrevInner() {
	// if out of alloc sectors
	if list.PrefixAllocSize == list.PrefixMapSize*arraySize {
		// make new data and copy to new
		toAdd := getNextSize(list.MapSize)
		newSize := toAdd + list.PrefixMapSize + list.MapSize
		newData := make([]*[arraySize]T, newSize)

		// alloc toAdd space before
		for i := 0; i < list.MapSize+list.PrefixMapSize; i++ {
			newData[i+toAdd] = list.Data[i]
		}

		list.Data = newData
		list.PrefixMapSize += toAdd
	}

	// alloc next sector
	list.Data[list.PrefixMapSize-list.PrefixAllocSize/arraySize-1] = GenerateInner[T]()
	list.PrefixAllocSize += arraySize
}

// consumes predicate, returns number of true instances.
func (list *Mappedlist[T]) Count(f func(T) bool) int {
	counts := make([]int, list.MapSize+list.PrefixMapSize)
	var wg sync.WaitGroup

	for ind, ptr := range list.Data {
		if ptr == nil {
			continue
		}
		wg.Add(1)
		// spawn goroutine for each inner array
		go func(index int, innerList [arraySize]T) {
			defer wg.Done()
			for _, elem := range innerList {
				if f(elem) {
					counts[index]++
				}
			}
		}(ind, *ptr)
	}

	// get sum of counts array
	wg.Wait()
	count := 0
	for _, num := range counts {
		count += num
	}
	return count
}

// does operation on each element in place. Consumed function must return same type as element
func (list *Mappedlist[T]) Map(f func(T) T) {
	var wg sync.WaitGroup
	for _, ptr := range list.Data {
		if ptr == nil {
			continue
		}
		wg.Add(1)
		go func(innerList *[arraySize]T) {
			defer wg.Done()
			for innerListInd, elem := range innerList {
				innerList[innerListInd] = f(elem)
			}
		}(ptr)
	}
	wg.Wait()
}

func (list *Mappedlist[T]) Append(element T) {
	// if all filled, allocate more space
	if list.Length == list.AllocSize {
		list.allocNextInner()
	}
	list.set(list.Length+list.PrefixLength, element)
	list.Length++
}

func (list *Mappedlist[T]) Prepend(element T) {
	// if all filled, allocate more space
	if list.PrefixLength == list.PrefixAllocSize {
		list.allocPrevInner()
	}
	list.PrefixLength++
	list.set(0, element)
}
