package zkadmin_test

import (
	"testing"
	"github.com/wvanbergen/kazoo-go"
	"github.com/stretchr/testify/assert"
)

func TestCreateTopic(t *testing.T) {

	config := kazoo.NewConfig()
	kz, err := kazoo.NewKazoo([]string{"127.0.0.1:2181"}, config)
	assert.NoError(t, err)

	err = kz.CreateTopic("test_ddd111213", 1, 1, make(map[string]interface{}))
	assert.NoError(t, err)

	err = kz.ChangeTopicConfig("test_ddd111213", map[string]interface{}{
		"a": "b",
	})
	assert.NoError(t, err)


	err = kz.DeleteTopic("test_ddd111213")
	assert.NoError(t, err)

}