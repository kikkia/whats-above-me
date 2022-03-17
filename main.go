package main

import "fmt"
import "encoding/json"
import "io/ioutil"
import "bytes"
import "net/http"
import "regexp"
import "math"
import "time"

// Represents the bounding box to watch
type Area struct {
	NW []float64 `json:"NW"`
	NE []float64 `json:"NE"`
	SW []float64 `json:"SW"`
	SE []float64 `json:"SE"`
}

type Config struct {
	Webhook string `json:"webhook_url"`
	Area Area `json:"area"`
	Airport string `json:"airport_code"`
}

type AircraftCoordinates struct {
	Coordinates []float64 `json:"coordinates"`
}

type Airport struct {
	Code string `json:"iata"`
}

type AircraftProperties struct {
	Id string `json:"flight_id"`
	Heading int `json:"direction"`
	Model string `json:"type"`
	FlightNumber string `json:"ident"`
	Icon string `json:"icon"`
	Origin Airport `json:"origin"`
	Destination Airport `json:"destination"`
	Type string `json:"flightType"`
	Altitude int `json:"altitude"`
	GroundSpeed int `json:"groundspeed"`
}

type Aircraft struct {
	Location AircraftCoordinates `json:"geometry"`
	Properties AircraftProperties `json:"properties"`
}

type VicinityResponse struct {
	AircraftList []Aircraft `json:"features"`
}

func main() {
	fmt.Println("Starting flight monitoring")
	config := GetConfig()

	seen := make(map[string]int64)

	for range time.Tick(time.Second * 20) {
		token := GetFAToken(config.Airport)
		aircraftAbove := GetAircraftInWatchArea(config.Area, token)
		for _, a := range aircraftAbove {
			now := time.Now()
			if val, ok := seen[a.Properties.FlightNumber]; ok {
				// 2 hr cooldown on post
				if (val + 7200 > now.Unix()) {
					continue
				}
			} 
			postToWebhook(a, config.Webhook)
			seen[a.Properties.FlightNumber] = now.Unix()
		}
    }
}

func GetConfig() Config {
	file, _ := ioutil.ReadFile("config.json")
	config := Config{}
	_ = json.Unmarshal([]byte(file), &config)
	return config
}

// Make an HTTP Get to flight aware to get a token for the call to get flights by coord
func GetFAToken(code string) string {
  url := fmt.Sprintf("https://flightaware.com/live/airport/%s", code)
  resp, err := http.Get(url)
  if err != nil {
	fmt.Println("Error getting token: ", err)
  }
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  
  r, _ := regexp.Compile("VICINITY_TOKEN\":\"([a-z0-9]+)\"")
  results := r.FindStringSubmatch(string(body))
  return results[1]
}

// Gets all aircraft in the given Area
func GetAircraftInWatchArea(area Area, token string) []Aircraft {
	// Calculate a bounding box around the area that are ints, we cannot pass floats to api
	maxLat := math.Max(area.NW[0], area.NE[0])
	minLat := math.Min(area.SW[0], area.SE[0])
	maxLon := math.Max(area.SE[1], area.NE[1])
	minLon := math.Max(area.NW[1], area.SW[1])

	aircraft := GetAircraftInVicinity(maxLon, minLon, maxLat, minLat, token)
	
	// iterate over all aircraft and return ones that are within the coords of the supplied area
	var inArea []Aircraft
	for _, a := range aircraft {
		if (a.Properties.Type == "airline" || a.Properties.Type == "cargo") {
			if (InArea(area, a)) {
				inArea = append(inArea, a)
			}
		}
	}
	return inArea
}

// Use a raycast to determine if we are within the polygon
func InArea(area Area, aircraft Aircraft) bool {
	// Iterate over all edges in area to check aircraft location
	// If odd number of intersects than we contain the aircraft
	// This is so ugly right now, ewwwww
	airLat := aircraft.Location.Coordinates[1]
	airLon := aircraft.Location.Coordinates[0]
	contained := raycastIntersect(airLat, airLon, area.NE[0], area.NE[1], area.NW[0], area.NW[1])
	// I really need to refactor this
	if (raycastIntersect(airLat, airLon, area.SE[0], area.SE[1], area.NE[0], area.NE[1])) {
		contained = !contained
	}
	if (raycastIntersect(airLat, airLon, area.SW[0], area.SW[1], area.SE[0], area.SE[1])) {
		contained = !contained
	}
	if (raycastIntersect(airLat, airLon, area.NW[0], area.NW[1], area.SW[0], area.SW[1])) {
		contained = !contained
	}
	return contained
}

func raycastIntersect(pLat float64, pLon float64, sLat float64, sLon float64, eLat float64, eLon float64) bool {
	if sLon > eLon {
		// Switch the points if otherwise.
		sLon, eLon = eLon, sLon
		sLat, eLat = eLat, sLat
	}

	// We need to ensure that the start point isnt on the test region
	for pLon == sLon || pLon == eLon {
		pLon = math.Nextafter(pLon, math.Inf(1))
	}

	// return false if we are outside
	if pLon < sLon || pLon > eLon {
		return false
	}

	if sLat > eLat {
		if pLat > sLat {
			return false
		}
		if pLat < eLat {
			return true
		}
	} else {
		if pLat > eLat {
			return false
		}
		if pLat < sLat {
			return true
		}
	}

	raySlope := (pLon - sLon) / (pLat - sLat)
	diagSlope := (eLon - sLon) / (eLat - sLat)

	return raySlope >= diagSlope
}


// Gets all aircraft in the given long, lat ranges
func GetAircraftInVicinity(maxLon float64, minLon float64, maxLat float64, minLat float64, token string) []Aircraft {
	url := fmt.Sprintf("https://flightaware.com/ajax/vicinity_aircraft.rvt?minLon=%f&minLat=%f&maxLon=%f&maxLat=%f&token=%s", minLon, minLat, maxLon, maxLat, token)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error getting token: ", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	vResp := VicinityResponse{}
	_ = json.Unmarshal([]byte(body), &vResp)
	return vResp.AircraftList
}

// post to webhook
func postToWebhook(aircraft Aircraft, hookUrl string) {	
	message := fmt.Sprintf("Look up! Thats flight %s from %s. It's a %s", aircraft.Properties.FlightNumber, aircraft.Properties.Origin.Code, aircraft.Properties.Model)
	postBody, _ := json.Marshal(map[string]string{
		"content": message,
	})
	requestBody := bytes.NewBuffer(postBody)

	resp, err := http.Post(hookUrl, "application/json", requestBody)
	if (err != nil) {
		fmt.Println("Failed to post to discord", err)
	}
    defer resp.Body.Close()
}