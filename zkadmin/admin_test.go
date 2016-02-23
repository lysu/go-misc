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

	topic := "test_d6"

	err = kz.CreateTopic(topic, 1, 1, make(map[string]interface{}))
	assert.NoError(t, err)

	err = kz.ChangeTopicConfig(topic, map[string]interface{}{
		"a": "b",
	})
	assert.NoError(t, err)

	err = kz.AddPartitions(topic, 2, "0:2", true)
	assert.NoError(t, err)

	err = kz.DeleteTopic(topic)
	assert.NoError(t, err)

}