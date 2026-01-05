// Example usage of the gibrun framework
// This demonstrates all core features: Gib, Run, and Sprint
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/arielfikru/gibrun"
)

// Program represents a government program (example struct)
type Program struct {
	Name   string `json:"name"`
	Budget int64  `json:"budget"`
	Target string `json:"target"`
}

func main() {
	// 1. Initialization (Pembentukan Kabinet)
	// Connect to Redis with transparent configuration
	app := gibrun.New(gibrun.Config{
		Addr:     "localhost:6379",
		Password: "", // Open transparency
		DB:       0,
	})
	defer app.Close()

	ctx := context.Background()

	// Check connection
	if err := app.Ping(ctx); err != nil {
		fmt.Printf("❌ Failed to connect to Redis: %v\n", err)
		fmt.Println("   Make sure Redis is running on localhost:6379")
		return
	}
	fmt.Println("✅ Connected to Redis!")

	// ========================================
	// 2. Giving Data (Gib) - Storing data
	// ========================================
	fmt.Println("\n📦 Demonstrating Gib (Store Data)...")

	makanSiang := Program{
		Name:   "Free Lunch Program",
		Budget: 400000000000,
		Target: "All Students",
	}

	// Store struct with TTL
	err := app.Gib(ctx, "program:priority").
		Value(makanSiang).
		TTL(5 * time.Minute).
		Exec()

	if err != nil {
		fmt.Printf("❌ Failed to store data: %v\n", err)
		return
	}
	fmt.Println("✅ Program stored successfully!")

	// Store a simple string
	err = app.Gib(ctx, "status:economy").
		Value("Positive Growth").
		TTL(10 * time.Minute).
		Exec()

	if err != nil {
		fmt.Printf("❌ Failed to store status: %v\n", err)
		return
	}
	fmt.Println("✅ Status stored successfully!")

	// ========================================
	// 3. Running Data (Run) - Retrieving data
	// ========================================
	fmt.Println("\n🏃 Demonstrating Run (Retrieve Data)...")

	var result Program
	found, err := app.Run(ctx, "program:priority").Bind(&result)

	if err != nil {
		fmt.Printf("❌ Error retrieving data: %v\n", err)
		return
	}

	if found {
		fmt.Printf("✅ Program found: %s\n", result.Name)
		fmt.Printf("   Budget: Rp %d\n", result.Budget)
		fmt.Printf("   Target: %s\n", result.Target)
	} else {
		fmt.Println("⚠️  Data not found (cache miss)")
	}

	// Retrieve simple string using Raw()
	status, found, err := app.Run(ctx, "status:economy").Raw()
	if err != nil {
		fmt.Printf("❌ Error retrieving status: %v\n", err)
		return
	}
	if found {
		fmt.Printf("✅ Economy Status: %s\n", status)
	}

	// ========================================
	// 4. Sprint - Atomic Operations
	// ========================================
	fmt.Println("\n⚡ Demonstrating Sprint (Atomic Operations)...")

	// Increment visitor counter
	count, err := app.Sprint(ctx, "counter:visitors").Incr()
	if err != nil {
		fmt.Printf("❌ Error incrementing: %v\n", err)
		return
	}
	fmt.Printf("✅ Visitor count: %d\n", count)

	// Increment by specific amount
	score, err := app.Sprint(ctx, "counter:score").IncrBy(100)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Score increased to: %d\n", score)

	// Decrement
	stock, err := app.Sprint(ctx, "counter:stock").Decr()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Stock count: %d\n", stock)

	// ========================================
	// Cleanup (optional)
	// ========================================
	fmt.Println("\n🧹 Cleaning up demo data...")
	app.Del(ctx, "program:priority", "status:economy", "counter:visitors", "counter:score", "counter:stock")
	fmt.Println("✅ Demo complete! 🏃💨🥛")
}
