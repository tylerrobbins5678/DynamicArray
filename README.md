# DynamicArray
Dynamic array implementation in GoLang with an extream focus on performance.
provides O((log(n) / k) time complexity for append, prepend
provides O(n + k*2) worst case space complexity, O(n) best case space complexity
filter and count functions provide O(k) time complexity and 0(1) space complexity

# Under the hood
This dynamic array allocates an golang slice of pointers to fixed size (k) arrays.
Doing so, the underlying element is never changed or moved within memory. The slice of pointers will reallocate fixed size arrays as needed to append and prepend.
filter and count functions are carried out my multiple go routines and consume a function that returns the altered value or a boolean to count.

# usage
list := mappedlist.Make\[int\]()
list -> []

list.Append(12345)
list -> [12345]

list.Prepend(1)
list -> [1,12345]

arr := list.ToArray()
arr -> [1,12345]

list = mappedlist.MakeFromArray\[int\](arr)
list -> [1,12345]

list.get(0) -> 1

list.set(0, 1000)
list -> [1000,12345]




