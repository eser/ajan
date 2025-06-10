package datafx

// This file contains shared types used across the datafx package.
// The main data access interfaces are now defined in connfx/data_ports.go
// as they represent the ports that connfx adapters must implement.

// User represents a basic user structure for examples.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Product represents a basic product structure for examples.
type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
}
