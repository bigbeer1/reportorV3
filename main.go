package reportorV3

import (
	"fmt"
	. "github.com/Monibuca/engine/v3"
	"github.com/go-redis/redis/v8"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ReportorConfig struct {
	MonibucaId string   // m7sId 唯一标识
	RedisHost  []string // redis地址
	RedisType  string   `default:",default=node,options=node|cluster"` // redis类型
	RedisPass  string   // redis密码

	EtcdHost        []string // etcd地址
	EtcdUsername    string   // etcd用户名
	EtcdPassword    string   // etcdPassword
	EtcdDialTimeout int64    `default:"10"` // 通讯超时时间  秒

	SyncServiceTime int64 `default:"30"`  // 同步服务器信息在线状态时间
	SyncTime        int64 `default:"30"`  // 同步阻塞时间
	SyncSaveTime    int64 `default:"180"` // 同步数据有效期时间

	redisCluster *redis.ClusterClient // redisCluster客户端
	redis        *redis.Client        // redis客户端

	etcd *clientv3.Client // etcd客户端

	MonibucaIp string // 用于设置MonibucaIp方便集群调度

	MonibucaPort string //用于设置MonibucaPort方便集群调度
}

type VideoChannel struct {
	StreamPath       string `json:"stream_path"`        // 流通道地址
	MonibucaId       string `json:"monibuca_id"`        // 服务器ID
	MonibucaIp       string `json:"monibuca_ip"`        // 服务器IP
	MonibucaPort     string `json:"monibuca_port"`      // 服务器Port
	StreamState      int64  `json:"stream_state"`       // 流状态
	StreamCreateTime int64  `json:"stream_create_time"` // 流拉取时间
	StreamType       string `json:"stream_type"`        // 流格式
}

var reportorConfig ReportorConfig

func init() {
	pc := PluginConfig{
		Name:   "Reportor",
		Config: &reportorConfig,
	}
	pc.Install(run)
}

func run() {
	fmt.Println(11)
}
