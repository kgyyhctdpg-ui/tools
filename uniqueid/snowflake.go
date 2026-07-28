package uniqueid

import (
	"log"
	"strconv"

	"github.com/sony/sonyflake"
)

var flake *sonyflake.Sonyflake

func init() {
	flake = sonyflake.NewSonyflake(sonyflake.Settings{})
}

func GenId() int64 {

	id, err := flake.NextID()
	if err != nil {
		log.Printf("flake NextID failed with %s \n", err)
		panic(err)
	}

	return int64(id)
}

func GenStrId() string {

	id, err := flake.NextID()
	if err != nil {
		log.Printf("flake NextID failed with %s \n", err)
		panic(err)
	}

	return strconv.Itoa(int(id))
}
