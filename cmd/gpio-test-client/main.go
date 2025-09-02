package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/dsyorkd/pi-controller/proto"
)

var (
	serverAddr = flag.String("addr", "localhost:9091", "Pi Agent server address")
	pin        = flag.Int("pin", 18, "GPIO pin number")
	value      = flag.Int("value", 1, "GPIO value to write (0 or 1)")
	command    = flag.String("cmd", "read", "Command to execute: read, write, config, pwm, list")
	frequency  = flag.Int("freq", 1000, "PWM frequency in Hz")
	dutyCycle  = flag.Int("duty", 50, "PWM duty cycle in percentage")
)

func main() {
	flag.Parse()

	// Connect to the Pi Agent
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Pi Agent: %v", err)
	}
	defer conn.Close()

	client := pb.NewPiAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch *command {
	case "health":
		testHealth(ctx, client)
	case "config":
		testConfigurePin(ctx, client, *pin)
	case "read":
		testReadPin(ctx, client, *pin)
	case "write":
		testWritePin(ctx, client, *pin, *value)
	case "pwm":
		testPWM(ctx, client, *pin, *frequency, *dutyCycle)
	case "list":
		testListPins(ctx, client)
	case "demo":
		runDemo(ctx, client)
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: health, config, read, write, pwm, list, demo")
	}
}

func testHealth(ctx context.Context, client pb.PiAgentServiceClient) {
	fmt.Println("Testing agent health...")
	
	resp, err := client.AgentHealth(ctx, &pb.AgentHealthRequest{})
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}

	fmt.Printf("Agent Health:\n")
	fmt.Printf("  Status: %s\n", resp.Status)
	fmt.Printf("  Version: %s\n", resp.Version)
	fmt.Printf("  GPIO Available: %t\n", resp.GpioAvailable)
	fmt.Printf("  Timestamp: %s\n", resp.Timestamp.AsTime().Format(time.RFC3339))
}

func testConfigurePin(ctx context.Context, client pb.PiAgentServiceClient, pin int) {
	fmt.Printf("Configuring pin %d as output...\n", pin)
	
	resp, err := client.ConfigureGPIOPin(ctx, &pb.ConfigureGPIOPinRequest{
		Pin:       int32(pin),
		Direction: pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT,
		PullMode:  pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE,
	})
	if err != nil {
		log.Printf("Configure failed: %v", err)
		return
	}

	fmt.Printf("Configuration result:\n")
	fmt.Printf("  Success: %t\n", resp.Success)
	fmt.Printf("  Message: %s\n", resp.Message)
	fmt.Printf("  Configured at: %s\n", resp.ConfiguredAt.AsTime().Format(time.RFC3339))
}

func testReadPin(ctx context.Context, client pb.PiAgentServiceClient, pin int) {
	fmt.Printf("Reading pin %d...\n", pin)
	
	resp, err := client.ReadGPIOPin(ctx, &pb.ReadGPIOPinRequest{
		Pin: int32(pin),
	})
	if err != nil {
		log.Printf("Read failed: %v", err)
		return
	}

	fmt.Printf("Pin %d value: %d (%s)\n", resp.Pin, resp.Value, 
		map[int32]string{0: "LOW", 1: "HIGH"}[resp.Value])
	fmt.Printf("Timestamp: %s\n", resp.Timestamp.AsTime().Format(time.RFC3339))
}

func testWritePin(ctx context.Context, client pb.PiAgentServiceClient, pin int, value int) {
	fmt.Printf("Writing value %d to pin %d...\n", value, pin)
	
	resp, err := client.WriteGPIOPin(ctx, &pb.WriteGPIOPinRequest{
		Pin:   int32(pin),
		Value: int32(value),
	})
	if err != nil {
		log.Printf("Write failed: %v", err)
		return
	}

	fmt.Printf("Write result:\n")
	fmt.Printf("  Pin: %d\n", resp.Pin)
	fmt.Printf("  Value: %d (%s)\n", resp.Value, 
		map[int32]string{0: "LOW", 1: "HIGH"}[resp.Value])
	fmt.Printf("  Timestamp: %s\n", resp.Timestamp.AsTime().Format(time.RFC3339))
}

func testPWM(ctx context.Context, client pb.PiAgentServiceClient, pin int, frequency int, dutyCycle int) {
	fmt.Printf("Setting PWM on pin %d (freq: %dHz, duty: %d%%)...\n", pin, frequency, dutyCycle)
	
	resp, err := client.SetGPIOPWM(ctx, &pb.SetGPIOPWMRequest{
		Pin:       int32(pin),
		Frequency: int32(frequency),
		DutyCycle: int32(dutyCycle),
	})
	if err != nil {
		log.Printf("PWM failed: %v", err)
		return
	}

	fmt.Printf("PWM result:\n")
	fmt.Printf("  Success: %t\n", resp.Success)
	fmt.Printf("  Message: %s\n", resp.Message)
	fmt.Printf("  Pin: %d\n", resp.Pin)
	fmt.Printf("  Frequency: %dHz\n", resp.Frequency)
	fmt.Printf("  Duty Cycle: %d%%\n", resp.DutyCycle)
	fmt.Printf("  Configured at: %s\n", resp.ConfiguredAt.AsTime().Format(time.RFC3339))
}

func testListPins(ctx context.Context, client pb.PiAgentServiceClient) {
	fmt.Println("Listing configured pins...")
	
	resp, err := client.ListConfiguredPins(ctx, &pb.ListConfiguredPinsRequest{})
	if err != nil {
		log.Printf("List failed: %v", err)
		return
	}

	if len(resp.Pins) == 0 {
		fmt.Println("No pins configured")
		return
	}

	fmt.Printf("Configured pins (%d):\n", len(resp.Pins))
	for i, pin := range resp.Pins {
		fmt.Printf("  %d. Pin %d:\n", i+1, pin.Pin)
		fmt.Printf("     Direction: %s\n", pin.Direction.String())
		fmt.Printf("     Pull Mode: %s\n", pin.PullMode.String())
		fmt.Printf("     Value: %d (%s)\n", pin.Value, 
			map[int32]string{0: "LOW", 1: "HIGH"}[pin.Value])
		fmt.Printf("     Last Updated: %s\n", pin.LastUpdated.AsTime().Format(time.RFC3339))
	}
}

func runDemo(ctx context.Context, client pb.PiAgentServiceClient) {
	fmt.Println("Running GPIO demo...")
	
	// Test health
	fmt.Println("\n1. Testing agent health...")
	testHealth(ctx, client)
	
	// Configure pin 18 as output
	fmt.Printf("\n2. Configuring pin %d as output...\n", *pin)
	testConfigurePin(ctx, client, *pin)
	
	// Write HIGH
	fmt.Printf("\n3. Writing HIGH to pin %d...\n", *pin)
	testWritePin(ctx, client, *pin, 1)
	
	// Read back
	fmt.Printf("\n4. Reading pin %d...\n", *pin)
	testReadPin(ctx, client, *pin)
	
	// Wait a bit
	fmt.Println("\n5. Waiting 2 seconds...")
	time.Sleep(2 * time.Second)
	
	// Write LOW
	fmt.Printf("\n6. Writing LOW to pin %d...\n", *pin)
	testWritePin(ctx, client, *pin, 0)
	
	// Read back
	fmt.Printf("\n7. Reading pin %d...\n", *pin)
	testReadPin(ctx, client, *pin)
	
	// List configured pins
	fmt.Println("\n8. Listing configured pins...")
	testListPins(ctx, client)
	
	// Test PWM (if supported)
	if *pin == 18 || *pin == 12 || *pin == 13 || *pin == 19 {
		fmt.Printf("\n9. Testing PWM on pin %d...\n", *pin)
		testPWM(ctx, client, *pin, 1000, 50)
		
		fmt.Println("\n10. Waiting 3 seconds for PWM...")
		time.Sleep(3 * time.Second)
		
		fmt.Printf("\n11. Stopping PWM (setting to 0%% duty cycle)...\n")
		testPWM(ctx, client, *pin, 1000, 0)
	}
	
	fmt.Println("\nDemo completed!")
}