package remiro

import (
	"testing"
)

func Test_redisHandler_HandleGET(t *testing.T) {

	t.Run(`[Given] a key is available in "destination" 
		    [When] a GET request for the key is received
		    [Then] GET and return the key value from "destination"`, func(t *testing.T) {

	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] the key is available in "source"
			 [And] deleteOnGet set to false
		    [When] a GET request for the key is received
		    [Then] GET and return the key value from "source"
		     [And] SET the value with the key to "destination"`, func(t *testing.T) {

	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] the key is available in "source"
			 [And] deleteOnGet set to true
		    [When] a GET request for the key is received
		    [Then] GET and return the key value from "source"
			 [And] SET the value with the key to "destination"
			 [And] DELETE the key from "source"`, func(t *testing.T) {

	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] the key is not available in "source"
		    [When] a GET request for the key is received
		    [Then] return error message from "source"`, func(t *testing.T) {

	})
}

func Test_redisHandler_HandleSET(t *testing.T) {

	t.Run(`[Given] deleteOnSet set to false
			[When] a SET request for a key is received
			[Then] SET the key with the value to "destination"`, func(t *testing.T) {

	})

	t.Run(`[Given] deleteOnSet set to true
			[When] a SET request for a key is received
			[Then] SET the key with the value to "destination"
			 [And] DELETE the key from "source"`, func(t *testing.T) {

	})
}

func Test_redisHandler_HandlePING(t *testing.T) {

	t.Run(`[When] a PING request is received
		   [Then] return PONG`, func(t *testing.T) {

	})
}

func Test_redisHandler_HandleDefault(t *testing.T) {

	t.Run(`[When] any request except GET, SET, and PING is received
		   [Then] forward the request to "destination"
		    [And] return the response`, func(t *testing.T) {

	})
}
