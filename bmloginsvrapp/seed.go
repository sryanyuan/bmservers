package main

var (
	gSeed = 0
)

func GetSeed() int {
	gSeed++
	return gSeed
}
