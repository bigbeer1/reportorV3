package reportorV3

import (
	"context"
	"encoding/json"
	"fmt"
	. "github.com/Monibuca/engine/v3"
	"github.com/Monibuca/utils/v3"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type ReportorConfig struct {
	MonibucaId string // m7sId 唯一标识
	RedisHost  string // redis地址
	RedisType  string `default:",default=node,options=node|cluster"` // redis类型
	RedisPass  string // redis密码

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

var p ReportorConfig

func init() {
	pc := PluginConfig{
		Name:   "Reportor",
		Config: &p,
	}
	pc.Install(run)
}

func run() {
	id := uuid.NewString()
	p.MonibucaId = id

	if len(p.RedisHost) > 0 {
		switch p.RedisType {
		case "node":
			//  单体redis
			p.redis = p.NewRedisManager()

		case "cluster":
			// 集群redis
			p.redisCluster = p.NewRedisClusterManager()

		}
	}

	if len(p.EtcdHost) > 0 {
		// etcd客户端
		p.etcd = p.NewEtcdManager()
	}

	// 同步服务器状态
	go p.SyncServiceWorker()
	go p.SyncWorker()
}

// 开启同步任务
func (p *ReportorConfig) SyncWorker() {
	// GB28181设备信息
	for {
		time.Sleep(time.Second * time.Duration(p.SyncTime))
		//p.SyncGBDevices()
		p.SyncVideoChannels()
	}

}

func (p *ReportorConfig) SyncServiceWorker() {
	for {
		// GB28181设备信息
		p.SyncService()
		time.Sleep(time.Second * time.Duration(p.SyncServiceTime))
	}

}

type M7sServiceInfo struct {
	StartTime time.Time //启动时间
	LocalIP   string
	Port      string
	Version   string
}

// 同步m7s服务端信息
func (p *ReportorConfig) SyncService() {
	key := fmt.Sprintf("m7sService:%v", p.MonibucaId)

	// 获取m7s 全局变量SysInfo  重新组装 这样容器中IP就不会存在这里了
	sysInfo := &M7sServiceInfo{StartTime: time.Now(), LocalIP: p.MonibucaIp, Port: p.MonibucaPort, Version: Version}

	data, err := json.Marshal(sysInfo)
	if err != nil {
		utils.Println(fmt.Sprintf("m7sService设备数据反序列化失败:%s", err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if p.redis != nil {
		cmd := p.redis.Set(ctx, key, data, time.Second*time.Duration(p.SyncSaveTime))
		if cmd.Err() != nil {
			utils.Println(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			return
		}
	}

	if p.redisCluster != nil {
		cmd := p.redisCluster.Set(ctx, key, data, time.Second*time.Duration(p.SyncSaveTime))
		if cmd.Err() != nil {
			utils.Println(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
			return
		}
	}

	if p.etcd != nil {
		leaseResp, err := p.etcd.Grant(ctx, p.SyncSaveTime)
		if err != nil {
			utils.Println(fmt.Sprintf("etcd创建lease失败:%s", err.Error()))
			return
		}
		// 写入键值对
		_, err = p.etcd.Put(ctx, key, string(data), clientv3.WithLease(leaseResp.ID))
		if err != nil {
			utils.Println(fmt.Sprintf("etcd存储失败:%s", err.Error()))
			return
		}
	}

}

// 同步流通道
func (p *ReportorConfig) SyncVideoChannels() {

	Streams.Range(func(stream *Stream) {
		publicKey := fmt.Sprintf("streamPath:%v", stream.StreamPath)
		privateKey := fmt.Sprintf("m7s:%v:streamPath:%v", p.MonibucaId, stream.StreamPath)

		videoChannel := &VideoChannel{
			StreamPath:       stream.StreamPath,
			MonibucaId:       p.MonibucaId,
			MonibucaIp:       p.MonibucaIp,
			MonibucaPort:     p.MonibucaPort,
			StreamState:      2,
			StreamCreateTime: stream.StartTime.UnixMilli(),
			StreamType:       stream.Type,
		}
		// 反序列化
		data, err := json.Marshal(videoChannel)
		if err != nil {
			utils.Println(fmt.Sprintf("SyncVideoChannel设备数据反序列化失败:%s", err.Error()))
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if p.redis != nil {
			cmd := p.redis.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				utils.Println(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return
			}

			cmd = p.redis.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				utils.Println(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return
			}
		}

		if p.redisCluster != nil {
			cmd := p.redisCluster.Set(ctx, publicKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				utils.Println(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return
			}

			cmd = p.redisCluster.Set(ctx, privateKey, data, time.Second*time.Duration(p.SyncSaveTime))
			if cmd.Err() != nil {
				utils.Println(fmt.Sprintf("redis数据同步失败:%s", cmd.Err().Error()))
				return
			}

		}

		if p.etcd != nil {
			leaseResp, err := p.etcd.Grant(ctx, p.SyncSaveTime)
			if err != nil {
				utils.Println(fmt.Sprintf("etcd创建lease失败:%s", err.Error()))
				return
			}
			// 写入键值对
			_, err = p.etcd.Put(ctx, publicKey, string(data), clientv3.WithLease(leaseResp.ID))
			if err != nil {
				utils.Println(fmt.Sprintf("etcd存储失败:%s", err.Error()))
				return
			}

			// 写入键值对
			_, err = p.etcd.Put(ctx, privateKey, string(data), clientv3.WithLease(leaseResp.ID))
			if err != nil {
				utils.Println(fmt.Sprintf("etcd存储失败:%s", err.Error()))
				return
			}
		}

		return
	})

}
