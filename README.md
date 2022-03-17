## What's above me
A simple go script to monitor a given area for aircraft and when an aircraft crosses into the area a notification is sent with basic flight info. 

### How to use
1. Create a file `config.json`
2. This file should contain the following fields: 
    - `webhook_url` - A url to a discord webhook
    - `airport_code` - An airport code, notifications are only sent for aircraft flying to the given airport
    - `area` - an object with 4 properties `NW`, `NE`, `SW`, `SE` with lat/long pairings. e.g:  
    ```
    "area" : {  
        "NW": [43.678457, -119.371461],  
        "NE": [43.676913, -119.277163],  
        "SW": [43.466555, -119.380597],  
        "SE": [43.461090, -119.271180]  
    }
    ```
    - `area` defines the bounding box used to detect flights.
3. Run the script with `go run main.go`

I moved somewhere directly below the approach vector or a busy airport. I was curious which flights I kept seeing over my head so I wrote this quick dirty script to monitor and let me know what they are and where they were from.