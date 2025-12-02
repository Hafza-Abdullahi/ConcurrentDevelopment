/** Project: Wa-tor Concurrency using Fork-Join parallelism
	Student Name: Hafza Abdullahi
	Student NUmber: C00286249

**/
package main

// IMPORTS
import (
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Constants
const (
	EMPTY = 0
	FISH  = 1
	SHARK = 2
)

// Structs

// Fish / shark 
type Agent struct {
	Type       int  // FISH or SHARK
	Age        int  // How many chronons
	StarveTime int  // Energy left, Shark only 
	Moved      bool // To prevent moving twice in one turn
}

// Simulation grid 
type World struct {
	Grid   [][]*Agent // 2D grid of pointers to Agents
	Size   int        // Grid size 
	Mutexs []sync.Mutex // One lock per row
}

// // variables 
var (
	numShark   = flag.Int("NumShark", 100, "Starting population of sharks")
	numFish    = flag.Int("NumFish", 200, "Starting population of fish")
	fishBreed  = flag.Int("FishBreed", 3, "Chronons before fish breed")
	sharkBreed = flag.Int("SharkBreed", 10, "Chronons before shark breed")
	starve     = flag.Int("Starve", 3, "Shark starve time")
	gridSize   = flag.Int("GridSize", 50, "Dimensions of world")
	threads    = flag.Int("Threads", 1, "Number of threads")
)




 // Main 
 // Set up the Fork-Join pattern and runs the loop
func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	w := initWorld(*gridSize)
	populateWorld(w)

	// Simulation Loop
	// will comment out visualisation for benchmarks to test speeds with higher threads
	chronon := 0
	
	start := time.Now()

	for chronon < 1000 { // Run for 1000 steps for results, or infinite for visuals
		
		// Reset 'Moved' flags
		resetMoves(w)

		// FORK: Process Grid in Chunks
		var wg sync.WaitGroup
		rowsPerThread := *gridSize / *threads

		for t := 0; t < *threads; t++ {
			startRow := t * rowsPerThread
			endRow := startRow + rowsPerThread
			if t == *threads-1 {
				endRow = *gridSize // Catch remainder
			}

			wg.Add(1)
			// Launch Thread (Goroutine)
			go func(s, e int) {
				defer wg.Done()
				processRows(w, s, e)
			}(startRow, endRow)
		}

		// JOIN: Wait for all threads to finish
		wg.Wait()

		// Visualization
		if *threads == 1  { 
			printWorld(w, chronon)
			time.Sleep(100 * time.Millisecond)
		}
		
		chronon++
	}

	elapsed := time.Since(start)
	fmt.Printf("Simulation took %s using %d threads\n", elapsed, *threads)
}

// Initialization

/**
 * @brief Initializes the World grid and synchronization primitives.
 * 
 * This function allocates memory individually, for the 2D grid structure and the array of Mutexes
 *
 * @param size The length and width of the square world (N x N).
 * @return *World A pointer to the fully initialized World struct.
 */
func initWorld(size int) *World {
	w := &World{
		Size:   size,
		Grid:   make([][]*Agent, size),
		Mutexs: make([]sync.Mutex, size),
	}
	// Iterate through each row to allocate the columns
	for i := range w.Grid {
		w.Grid[i] = make([]*Agent, size)
	}
	return w
}
/**
 * @brief Populates the world with the initial distribution of Fish and Sharks.
 * 
 * Uses the global flags (*numFish, *numShark) to determine counts.
 * Agents are placed at random coordinates using the math/rand generator.
 * If a random coordinate is already occupied, agent is skipped for that iteration 
 * to prevent overwriting.
 *
 * @param w Pointer to the World object to be populated.
 */
func populateWorld(w *World) {
	// Add Fish
	for i := 0; i < *numFish; i++ {
		r, c := rand.Intn(w.Size), rand.Intn(w.Size)
		if w.Grid[r][c] == nil {
			w.Grid[r][c] = &Agent{Type: FISH}
		}
	}
	// Add Sharks
	for i := 0; i < *numShark; i++ {
		r, c := rand.Intn(w.Size), rand.Intn(w.Size)
		if w.Grid[r][c] == nil {
			w.Grid[r][c] = &Agent{Type: SHARK, StarveTime: *starve}
		}
	}
}

/**
 * @brief Resets the 'Moved' status flag for every agent in the grid.
 * 
 * This function is called at the beginning of every Chronon.
 * It ensures that agents moved in the previous step are eligible to move again 
 * in the current step. Without this, agents would freeze after one move
 *
 * @param w Pointer to the World object.
 */

func resetMoves(w *World) {
	for r := 0; r < w.Size; r++ {
		for c := 0; c < w.Size; c++ {
			if w.Grid[r][c] != nil {
				w.Grid[r][c].Moved = false
			}
		}
	}
}

// logic for pattern

/**
 * @brief Process a strip of rows
 */
func processRows(w *World, startRow, endRow int) {
	for r := startRow; r < endRow; r++ {
		for c := 0; c < w.Size; c++ {
			
			// lock the row to read safely and its neighbours
			// simply, only lock when WRITE (Move).
			
			agent := w.Grid[r][c]

			// Skip if the spot is empty
			// or, if this creature has already moved during this time step
			if agent == nil || agent.Moved {
				continue
			}

			if agent.Type == SHARK {
				updateShark(w, r, c, agent)
			} else {
				updateFish(w, r, c, agent)
			}
		}
	}
}

/**
 * @brief Runs the logic for a singular Fish.
 * Rules: Move randomly. If old enough, leave a baby behind.
 */
func updateFish(w *World, r, c int, fish *Agent) {
	fish.Age++
	// Find empty neighbors
	candidates := getNeighbors(w, r, c, EMPTY)
	
	if len(candidates) > 0 {
		// Pick random move
		dest := candidates[rand.Intn(len(candidates))]
		
		// Breed logic
		if fish.Age >= *fishBreed {
			fish.Age = 0
			// Leave a baby fish behind, move the parent
			moveAgent(w, r, c, dest.r, dest.c, fish, true) 
		} else {
			// or move
			moveAgent(w, r, c, dest.r, dest.c, fish, false)
		}
	}
	fish.Moved = true
}

/**
 * @brief Runs the logic for a single Shark
 * Rules: if energy is lost creature dies. Else, hunt the fish. or, move randomly.
 */

func updateShark(w *World, r, c int, shark *Agent) {
	shark.Age++
	shark.StarveTime--

	// Check for death
	if shark.StarveTime < 0 {
		w.Mutexs[r].Lock()
		w.Grid[r][c] = nil // Die
		w.Mutexs[r].Unlock()
		return
	}

	// Hunt Fish
	fishNeighbors := getNeighbors(w, r, c, FISH)
	if len(fishNeighbors) > 0 {
		dest := fishNeighbors[rand.Intn(len(fishNeighbors))]
		shark.StarveTime = *starve // Reset energy
		
		// Breed Logic
		spawn := false
		if shark.Age >= *sharkBreed {
			shark.Age = 0
			spawn = true
		}
		moveAgent(w, r, c, dest.r, dest.c, shark, spawn)
		shark.Moved = true
		return
	}

	// Movement Check (Priority #2)
	// If no food was found, look for empty water just like a fish does.
	emptyNeighbors := getNeighbors(w, r, c, EMPTY)
	if len(emptyNeighbors) > 0 {
		dest := emptyNeighbors[rand.Intn(len(emptyNeighbors))]
		
		spawn := false
		if shark.Age >= *sharkBreed {
			shark.Age = 0
			spawn = true
		}
		moveAgent(w, r, c, dest.r, dest.c, shark, spawn)
	}
	shark.Moved = true
}

// Helper funcs 

type Point struct { r, c int }

// Toroidal wrap helper
func wrap(val, max int) int {
	if val < 0 { return max - 1 }
	if val >= max { return 0 }
	return val
}

func getNeighbors(w *World, r, c int, targetType int) []Point {
	// Look N, E, S, W
	dr := []int{-1, 0, 1, 0}
	dc := []int{0, 1, 0, -1}
	var matches []Point

	for i := 0; i < 4; i++ {
		nr := wrap(r+dr[i], w.Size)
		nc := wrap(c+dc[i], w.Size)

		// Look at neighbor
		neighbor := w.Grid[nr][nc]

		// Check if this specific neighbor matches the criteria 

		if targetType == EMPTY && neighbor == nil {
			matches = append(matches, Point{nr, nc})
		} else if targetType == FISH && neighbor != nil && neighbor.Type == FISH {
			matches = append(matches, Point{nr, nc})
		}
	}
	return matches
}

// Critical Section: Moving an agent
func moveAgent(w *World, fromR, fromC, toR, toC int, agent *Agent, leaveChild bool) {
	// To prevent deadlocks or race conditions, lock the lowest row index first
	first, second := fromR, toR
	if first > second {
		first, second = second, first
	}
	
	w.Mutexs[first].Lock()
	if first != second {
		w.Mutexs[second].Lock()
	}

	// Double check target is still valid 
	// If it was a hunt, check it's still a fish or empty. 
	// the move is forced
	
	// Perform Move
	w.Grid[toR][toC] = agent
	
	if leaveChild {
		// Create new agent at old spot
		child := &Agent{Type: agent.Type, Age: 0}
		if agent.Type == SHARK { child.StarveTime = *starve }
		w.Grid[fromR][fromC] = child
	} else {
		w.Grid[fromR][fromC] = nil
	}

	if first != second {
		w.Mutexs[second].Unlock()
	}
	w.Mutexs[first].Unlock()
}

// Graphical output

func printWorld(w *World, chronon int) {
	// Move cursor to top left (ANSI)
	fmt.Print("\033[H") 
	fmt.Printf("Chronon: %d | Fish: Green, Shark: Red\n", chronon)
	
	// Simple buffer string to avoid flickering IO
	output := ""
	for r := 0; r < w.Size; r++ {
		for c := 0; c < w.Size; c++ {
			cell := w.Grid[r][c]
			if cell == nil {
				output += ". "
			} else if cell.Type == FISH {
				output += "\033[32m><\033[0m" // Green Fish
			} else if cell.Type == SHARK {
				output += "\033[31m^^\033[0m" // Red Shark
			}
		}
		output += "\n"
	}
	fmt.Print(output)
}