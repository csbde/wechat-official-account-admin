package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ==========================================================
// 微信错误码友好化: 把常见的凭据/权限类错误翻译成可操作的中文说明
// 目的: 避免用户只看到 "[48001] api unauthorized" 这类无法自助排查的原文
// ==========================================================

var wxErrcodeRe = regexp.MustCompile(`\[(\d+)\]`)

// wxErrorHints 常见错误码 -> 处置建议
var wxErrorHints = map[int]string{
	40001: "access_token 无效或已过期。多为 AppSecret 填写错误、公众号/小程序换过密钥，或 IP 未加入白名单，请在「公众号与开放平台」核对凭据后重试。",
	40013: "AppID 不合法。请检查该账号的 AppID 是否填写正确、是否已在微信侧完成接入配置。",
	40164: "调用方 IP 未加入白名单。请在微信公众平台「设置与开发 -> 基本配置 -> IP 白名单」中添加本服务器出口 IP (「系统与权限」页面可查看当前出口 IP)，或改用第三方平台授权托管 (免白名单)。",
	41001: "缺少 access_token。该账号尚未成功取得调用凭据，请先检查凭据与网络。",
	42001: "access_token 已超时。系统会自动刷新，请稍后重试；持续出现请核对服务器时间与时区。",
	48001: "该接口功能未授权 (api unauthorized)。此类接口属于微信开放平台「第三方平台代调用」能力，只有扫码授权托管的账号才能调用；「直连凭据」账号请改用「微信扫码一键授权接入」重新托管 (并勾选对应权限集)，或到微信公众平台/小程序后台手动操作。",
	48002: "接口权限不足。请在微信开放平台后台为该第三方平台勾选对应权限集，并让公众号/小程序重新授权。",
	61003: "该小程序为第三方平台代开发小程序，必须使用授权方 (authorizer) 凭据调用，请确认账号已通过扫码授权接入。",
	61004: "授权方 appid 与第三方平台不匹配，请在「公众号与开放平台」中重新完成扫码授权。",
	61005: "授权已失效或被取消，请让管理员重新扫码授权后再操作。",
	89503: "该操作需要公众号/小程序管理员在微信中确认，请在管理员确认后重试。",
	45009: "达到接口每日调用上限，请明日再试或减少调用频次。",
	40037: "template_id 无效：这个模板 ID 不属于当前调用凭据对应的公众号 (或该模板已被微信删除)。请确认顶部选中的公众号就是模板所属账号；到「消息模板」刷新模板库后重新选择；注意「改过所属行业会清空原行业模板」、「订阅消息的 priTmplId」与「公共模板编号 template_id_short」都不能当作模板消息的 template_id 使用。",
	40003: "invalid openid：接收者 openid 不属于当前公众号。请改用「粉丝与标签」里该账号下的 openid (不同公众号/小程序的 openid 不通用)。",
	47003: "data 参数与模板关键词不匹配：缺少必填关键词、关键词名写错，或内容超出长度限制。请刷新模板库后按最新关键词逐项填写。",
}

// FriendlyWxError 为微信错误补充中文处置建议，已是友好提示或无法识别错误码时原样返回
func FriendlyWxError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "处置建议") || strings.Contains(msg, "请在「公众号与开放平台」") {
		return err
	}
	m := wxErrcodeRe.FindStringSubmatch(msg)
	if m == nil {
		return err
	}
	code := 0
	if _, e := fmt.Sscanf(m[1], "%d", &code); e != nil {
		return err
	}
	hint, ok := wxErrorHints[code]
	if !ok {
		return err
	}
	return fmt.Errorf("%s — 处置建议: %s", msg, hint)
}

// ErrThirdPartyOnly 第三方平台代调用专属能力被直连凭据账号调用时返回
var ErrThirdPartyOnly = errors.New("该能力仅支持微信开放平台第三方授权托管的账号")
