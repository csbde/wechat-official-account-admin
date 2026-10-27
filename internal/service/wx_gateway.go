package service

import (
	"context"
	"encoding/json"
	"encoding/xml"
		"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"go.uber.org/zap"
)

type WxBaseMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        int64    `xml:"MsgId"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
	Ticket       string   `xml:"Ticket"`
}

type WxTextReply struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
}

func HandleWxMessage(ctx context.Context, appid string, rawXml []byte) ([]byte, error) {
	var msg WxBaseMessage
	if err := xml.Unmarshal(rawXml, &msg); err != nil {
		return nil, err
	}

	logger.Info("Received WeChat message",
		zap.String("appid", appid),
		zap.String("from", msg.FromUserName),
		zap.String("msg_type", msg.MsgType),
		zap.String("event", msg.Event),
		zap.String("content", msg.Content),
	)

	// 处理事件
	if msg.MsgType == "event" {
		switch strings.ToLower(msg.Event) {
		case "subscribe":
			// 记录或更新关注者
			go recordFanSubscribe(appid, msg.FromUserName, msg.EventKey)
			return matchAndBuildReply(appid, "subscribe", "", msg.FromUserName, msg.ToUserName)
		case "unsubscribe":
			go recordFanUnsubscribe(appid, msg.FromUserName)
			return nil, nil
		case "click":
			return matchAndBuildReply(appid, "keyword", msg.EventKey, msg.FromUserName, msg.ToUserName)
		}
	}

	// 处理文本消息自动回复
	if msg.MsgType == "text" {
		reply, err := matchAndBuildReply(appid, "keyword", strings.TrimSpace(msg.Content), msg.FromUserName, msg.ToUserName)
		if err == nil && reply != nil {
			return reply, nil
		}
		// 默认回复
		return matchAndBuildReply(appid, "default", "", msg.FromUserName, msg.ToUserName)
	}

	return nil, nil
}

func recordFanSubscribe(appid, openid, eventKey string) {
	var fan model.WxFan
	err := database.DB.Where("app_id = ? AND open_id = ?", appid, openid).First(&fan).Error
	if err != nil {
		fan = model.WxFan{
			AppID:          appid,
			OpenID:         openid,
			Subscribe:      1,
			SubscribeTime:  time.Now().Unix(),
			SubscribeScene: eventKey,
		}
		database.DB.Create(&fan)
	} else {
		database.DB.Model(&fan).Updates(map[string]interface{}{
			"subscribe":      1,
			"subscribe_time": time.Now().Unix(),
		})
	}
}

func recordFanUnsubscribe(appid, openid string) {
	database.DB.Model(&model.WxFan{}).
		Where("app_id = ? AND open_id = ?", appid, openid).
		Update("subscribe", 0)
}

func matchAndBuildReply(appid, ruleType, content, toUser, fromUser string) ([]byte, error) {
	var rules []model.WxAutoReply
	query := database.DB.Where("app_id = ? AND enabled = ?", appid, true)

	if ruleType == "subscribe" || ruleType == "default" {
		query = query.Where("rule_type = ?", ruleType)
	} else {
		query = query.Where("rule_type = 'keyword'")
	}

	if err := query.Find(&rules).Error; err != nil || len(rules) == 0 {
		return nil, nil
	}

	var matched *model.WxAutoReply
	if ruleType == "subscribe" || ruleType == "default" {
		matched = &rules[0]
	} else {
		for _, r := range rules {
			var kws []struct {
				Keyword   string `json:"keyword"`
				MatchType string `json:"match_type"`
			}
			if err := json.Unmarshal([]byte(r.Keywords), &kws); err == nil {
				for _, kw := range kws {
					if kw.MatchType == "full" && kw.Keyword == content {
						matched = &r
						break
					} else if kw.MatchType == "half" && strings.Contains(content, kw.Keyword) {
						matched = &r
						break
					}
				}
			}
			if matched != nil {
				break
			}
		}
	}

	if matched == nil {
		return nil, nil
	}

	if matched.ReplyType == "text" {
		reply := WxTextReply{
			ToUserName:   toUser,
			FromUserName: fromUser,
			CreateTime:   time.Now().Unix(),
			MsgType:      "text",
			Content:      matched.Content,
		}
		return xml.Marshal(reply)
	}

	return nil, nil
}
