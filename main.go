package main 
    
import (
	"fmt"
	"encoding/json"
	"log"
	"net/http"
  "github.com/google/uuid"
  "slices"
  "regexp"
  "strconv"
  "math"
  "strings"
)

// Process Receipt response
type Id struct {
	Id string `json:"id"`
}

// Get Points response
type Points struct {
  Points int `json:"points"`
}

type Item struct {
  ShortDescription string `json:"shortDescription"`
  Price string `json:"price"`
}

type Receipt struct {
  Retailer string `json:"retailer"`
  PurchaseDate string `json:"purchaseDate"`
  PurchaseTime string `json:"purchaseTime"`
  Items []Item
  Total string `json:"total"`
}

// Struct for local storage
type ReceiptArray struct {
  Id string `json:"id"`
  Receipt Receipt
}
var receipts []ReceiptArray // Locally store receipts in an array

// Process Receipts API. Generates an id for the given receipt and stores the receipt in the global recaipts array
func processReceipts(w http.ResponseWriter, r *http.Request) {
  //Only allow post methods. Return 405 if a different method
  if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

  // Process the request body and put it into a struct
  var receipt Receipt
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&receipt)

  // Return 400 if the request body is not formatted properly
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

  // Generate new id
	newUUID := uuid.New()
  id := Id{Id: newUUID.String()}

  // Store Receipt info into global array
  newReceipt := ReceiptArray{Id: newUUID.String(), Receipt: receipt}
  receipts = append(receipts, newReceipt)

  // Respond with 200 and the new id
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(id)
}

// Get Points API. Uses the given id to retrieve the recaipt's information from the receipt array
func getPoints(w http.ResponseWriter, r *http.Request) {
  //Only allow GET methods. Return 405 if a different method
  if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

  // Get the Receipt struct with given receiptId
  id := r.PathValue("id")
  rec := getReceipt(id)

  // Total up all of the points
  totalPoints := AlphanumericPoints(rec.Receipt.Retailer) + roundDollarPoints(rec.Receipt.Total) + multpleOf25Points(rec.Receipt.Total) + everyTwoItemPoints(rec.Receipt.Items) + trimmedDescPoints(rec.Receipt.Items) + dateOddPoints(rec.Receipt.PurchaseDate) + timePoints(rec.Receipt.PurchaseTime)
  result := Points{Points: totalPoints}

  // Respond with 200 and the number of points awarded
  w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// Given a uuid, get the corresponding receipt from the receipt array
func getReceipt(id string) ReceiptArray {
  idx := slices.IndexFunc(receipts, func(c ReceiptArray) bool { return c.Id == id })
  return receipts[idx]
}

// Returns the number of alphanumeric characters in the retailer name
func AlphanumericPoints(Retailer string) int {
  reg, err := regexp.Compile("[^a-zA-Z0-9]+") 
	if err != nil {
		panic(err)
	}

  // Replace every non alphanumeric character with empty string
	newString := reg.ReplaceAllString(Retailer, "")
  return len(newString) // return length
}

// Returns 50 if the total is a full dollar, returns 0 if not
func roundDollarPoints(total string) int {
  floatTotal, err := strconv.ParseFloat(total, 8) // Parse total to float
  if err != nil {
		panic(err)
	}
  
  if floatTotal == math.Trunc(floatTotal) { // If truncating the decimals equals the original total, it is a full dollar
    return 50
  } else {
    return 0
  }
}

// Returns 25 if the total is a multiple of .25, returns 0 if not
func multpleOf25Points(total string) int {
  floatTotal, err := strconv.ParseFloat(total, 8) // Parse total to float
  if err != nil {
		panic(err)
	}

  multBy4 := floatTotal * 4 // Multiply total by 4
  if multBy4 == math.Floor(multBy4) { // Only multiples of .25 * 4 will be a round integer, so it should be the same whether it is rounded down or not.
    return 25
  } else {
    return 0
  }
}

// For every 2 items, add 5 points
func everyTwoItemPoints(Items []Item) int {
  halfItems := len(Items) / 2
  return halfItems * 5
}

// If the trimmed length of the item description is a multiple of 3, 
// multiply the price by 0.2 and round up to the nearest integer. 
// The result is the number of points earned.
func trimmedDescPoints(Items []Item) int {
  points := 0
  for i := 0; i < len(Items); i++ { // Loop through items
    trimStr := strings.TrimSpace(Items[i].ShortDescription) // trim the description
    if len(trimStr) % 3 == 0 { // if the trimmed description is a multiple of 3, multiply the price by 0.2, round up, and add it to the total points
      price, _ := strconv.ParseFloat(Items[i].Price, 8)
      points = points + int(math.Ceil(price * 0.2))
    }
  }
  return points
}

// Returns 6 if the date is odd
func dateOddPoints(date string) int {
  day, _ := strconv.ParseInt(date[len(date)-3:], 10, 64) // Get the last two characters of the date string and convert to int to get the day
  if day % 2 == 0 { // If the day is a multiple of 2, return 0. If not a multiple of 2, return 6
    return 0
  } else {
    return 6
  }
}

// Returns 10 if the time is after 2pm and before 4 pm (I have including 2:00 and excluding 4:00, option below to exclude 2:00 if desired)
func timePoints(time string) int {
  hour, _ := strconv.ParseInt(time[0:2], 10, 64) // Get the first two characters of the time string and convert to int to get the hour
  // min, _ := strconv.ParseInt(time[len(time)-2:], 10, 64)
  
  if hour >= 14 && hour < 16 { // Change if to `(hour > 14 && hour < 16) || (hour == 14 && min > 00)` and uncomment the line above, if you want to exclude 2:00pm exaclty
    return 10
  } else {
    return 0
  }
}

// Handle API requests and listen to 8080
func handleRequests() {
	http.Handle("/receipts/process", http.HandlerFunc(processReceipts))
  http.Handle("/receipts/{id}/points", http.HandlerFunc(getPoints))
	log.Fatal(http.ListenAndServe(":8080", nil))
  }
    
// Main function 
func main() { 
    fmt.Println("Listening on localHost:8080...") 
    handleRequests()
} 
