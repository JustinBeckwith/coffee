package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/JustinBeckwith/go-yelp/yelp"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/guregu/null"
)

func main() {

	// set up the server
	r := gin.Default()
	r.Use(static.Serve("/", static.LocalFile("public", false)))
	r.LoadHTMLGlob("templates/*")

	// index route
	r.GET("/", func(c *gin.Context) {
		obj := gin.H{"titler": "Main website"}
		c.HTML(http.StatusOK, "index.tmpl", obj)
	})

	r.GET("/GetCoffee", func(c *gin.Context) {
		q := c.Request.URL.Query()

		// get the latitude from the query string
		strlat := q.Get("lat")
		lat, err := strconv.ParseFloat(strlat, 64)
		if err != nil {
			c.JSON(400, gin.H{
				"status": "failed",
				"error":  "invalid latitude",
			})
			return
		}

		// get the longitude from the query string
		strlon := q.Get("lon")
		lon, err := strconv.ParseFloat(strlon, 64)
		if err != nil {
			c.JSON(400, gin.H{
				"status": "failed",
				"error":  "invalid longitude",
			})
			return
		}

		// get the yelp API keys from config or environment
		options, err := getOptions()
		if err != nil {
			c.JSON(400, gin.H{
				"status": "failed",
				"error":  "unable to query yelp api",
			})
			return
		}

		// create a new yelp client with the auth keys
		client := yelp.New(options, nil)

		// search for all coffee near the given coordinates
		searchOptions := yelp.SearchOptions{
			GeneralOptions: &yelp.GeneralOptions{
				Term: "coffee",
			},
			CoordinateOptions: &yelp.CoordinateOptions{
				Latitude:         null.FloatFrom(lat),
				Longitude:        null.FloatFrom(lon),
				Accuracy:         null.FloatFromPtr(nil),
				Altitude:         null.FloatFromPtr(nil),
				AltitudeAccuracy: null.FloatFromPtr(nil),
			},
		}
		results, err := client.DoSearch(searchOptions)
		if err != nil {
			c.JSON(400, gin.H{
				"status": "failed",
				"error":  "unable to query yelp api",
			})
			return
		}

		// Everything worked!  Return the JSON from the yelp API to the client.
		c.JSON(http.StatusOK, results)
	})

	// run the server
	r.Run(":8080")
}

// getOptions obtains the keys required to use the Yelp API from a config file
// or from environment variables.
func getOptions() (options *yelp.AuthOptions, err error) {

	var o *yelp.AuthOptions

	// start by looking for the keys in config.json
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		// if the file isn't there, check environment variables
		o = &yelp.AuthOptions{
			ConsumerKey:       os.Getenv("CONSUMER_KEY"),
			ConsumerSecret:    os.Getenv("CONSUMER_SECRET"),
			AccessToken:       os.Getenv("ACCESS_TOKEN"),
			AccessTokenSecret: os.Getenv("ACCESS_TOKEN_SECRET"),
		}
		if o.ConsumerKey == "" || o.ConsumerSecret == "" || o.AccessToken == "" || o.AccessTokenSecret == "" {
			return o, errors.New("to use the sample, keys must be provided either in a config.json file at the root of the repo, or in environment variables")
		}
	} else {
		err = json.Unmarshal(data, &o)
		return o, err
	}
	return o, nil
}
