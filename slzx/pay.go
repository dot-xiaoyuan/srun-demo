package slzx

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dot-xiaoyuan/srun-demo/floger"
	"github.com/dot-xiaoyuan/srun-demo/helper"
	"github.com/jinzhu/gorm"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ServiceUrl    = "https://cyg.shang-lian.com/"
	TransferMerId = "111261"                           // 商户编号
	TransferItem  = "T10100000000383"                  // 转账项目编号
	Md5Key        = "qknalbp3dpq95akriibk8xomk1zlhi16" // md5签名串
	PlatId        = "55555_md5"                        // 测试平台编号
	NotifyUrl     = "https://yuantong.pro"
	Key           = "yczBqtTa"
)

type RespData struct {
	PayFee       string `json:"payfee"`
	TransferList []struct {
		TransferItem string `json:"transferItem"`
	} `json:"transferList"`
	MerChantOrderId string `json:"merchantorderid"`
	ReturnMsg       string `json:"returnmsg"`
	Reserved        string `json:"reserved"`
	Postscript      string `json:"postscript"`
	ReturnCode      string `json:"returncode"`
	OrderFee        string `json:"orderfee"`
	ChannelSerialNo string `json:"channelserialno"`
	Status          string `json:"status"`
}

type QueryResponse struct {
	TransDate      string   `json:"trans_date"`
	CreateTime     string   `json:"create_time"`
	AccountDate    string   `json:"account_date"`
	DiscountAmount float64  `json:"discount_amount"`
	ReturnCode     string   `json:"returncode"`
	ReturnType     string   `json:"returntype"`
	Sign           string   `json:"sign"`
	RespData       string   `json:"respdata"`
	DealAmount     float64  `json:"deal_amount"`
	OverAmount     float64  `json:"over_amount"`
	SLBusIId       string   `json:"slbusiid"`
	ReturnMsg      string   `json:"returnmsg"`
	PresetId       string   `json:"preset_id"`
	OrderAmount    float64  `json:"order_amount"`
	Paid           string   `json:"paid"`
	ResponseTime   string   `json:"responsetime"`
	ReceiveAmount  float64  `json:"receive_amount"`
	ResData        RespData `json:"resdata"`
}

type PayModel struct {
	Username    string `json:"phone" form:"phone"`
	OutTradeNo  string `json:"order_id" form:"order_id"`
	Money       string `json:"price" form:"price"`
	ProductId   int    `json:"product_id" form:"product_id"`
	ProductName string `json:"product_name" form:"product_name"`
	PayMethod   string `json:"pay_method" form:"pay_method"`
	BuyTime     int64  `json:"buy_time"`
	Status      string `json:"status"`
	Payment     string `json:"payment"`
	PayType     string `json:"pay_type"`
	Remark      string `json:"remark"`
	Mobile      string `json:"mobile" form:"mobile"`
	PackageId   string `json:"package_id" form:"package_id"`
	SyncUrl     string `json:"sync_url" form:"sync_url"`
	ClientIP    string `json:"client_ip"`
	Email       string `json:"email" form:"email"`
}

type Pay struct {
	Model PayModel
	Conn  *gorm.DB
}

type PayResponse struct {
	ReturnCode     string `json:"returncode"`
	ReturnMsg      string `json:"returnmsg"`
	ReturnType     string `json:"returntype"`
	SLIBusIId      string `json:"slibusiid"`
	ResponstreTime string `json:"responstretime"`
	RespData       string `json:"respdata"`
	Data           string `json:"data"`
	Sign           string `json:"sign"`
}

func (p *Pay) UnifiedOrder() (res interface{}, err error) {
	p.Model.PayType = "19"
	res, err = p.NativePayRequest()

	//if err = db.AddOrder(p.Model, p.Conn); err != nil {
	//	floger.Errorf("AddOrder err: %s", err.Error())
	//	return res, err
	//}

	return
}

// NativePayRequest 扫码支付
func (p *Pay) NativePayRequest() (res PayResponse, err error) {
	// 接口地址
	nativePayRequestUrl := fmt.Sprintf("%s%s", ServiceUrl, "sltf-outside/inter/nativePayRequest")
	// code type
	codeType := "PAYLINK"

	merchantorderid := p.Model.OutTradeNo

	// 组装业务参数
	var transferList []map[string]interface{}
	transferItemMap := make(map[string]interface{})
	transferItemMap["transferMerId"] = TransferMerId
	transferItemMap["transferItem"] = TransferItem
	transferList = append(transferList, transferItemMap)

	reqData := map[string]interface{}{
		"txtype":          "03",
		"transferList":    transferList,
		"useridType":      "0",
		"code_type":       codeType,
		"userid":          "18510299945",
		"username":        "张三",
		"device_info":     "temp",
		"merchantorderid": merchantorderid,
		"orderfee":        "1", // 订单金额(分)
		"receiveurl":      NotifyUrl,
	}

	reqDataJson, _ := json.Marshal(reqData)

	floger.Debug5("des 加密前的数据reqDataJson:", string(reqDataJson))

	reqDataDes, _ := EncryptDES(Key, string(reqDataJson))

	floger.Debug5("des 加密后的数据reqDataDes:", reqDataDes)

	signContent := reqDataDes + "&key=" + Md5Key

	floger.Debug5("md5 签名前 signContent:", signContent)

	signMsg := strings.ToLower(helper.Md5(signContent))

	floger.Debug5("md5 签名后 signMsg:", signMsg)

	headMap := map[string]interface{}{
		"version":         "V1.0",
		"charset":         "1",
		"platid":          PlatId,
		"produid":         "PM2000",
		"channelserialno": p.Model.OutTradeNo,
		"channeltime":     time.Now().Format("20060102150405"),
		"reqdata":         reqDataDes,
		"sign":            signMsg,
	}

	sendJson, _ := json.Marshal(headMap)

	floger.Debug5("发送数据 sendJson:", string(sendJson))
	resp, err := doRequest(nativePayRequestUrl, http.MethodPost, sendJson)
	if err != nil {
		floger.Error("Failed to request slzx, err:", err)
		return res, err
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		floger.Error("Failed to json_decode response:", err)
		return res, err
	}
	floger.Debug5("返回结果 resultStr:", res)
	// 解密
	if res.ReturnCode != "0000" {
		return res, errors.New(res.ReturnMsg)
	}
	res.Data, _ = DecryptDES(Key, res.RespData)

	return res, nil
}

func (p *Pay) OrderQuery() (res QueryResponse, err error) {
	// 接口地址
	nativePayRequestUrl := fmt.Sprintf("%s%s", ServiceUrl, "sltf-outside/inter/pmManageOrderQuery")

	reqData := map[string]interface{}{
		"txtype":             "03",
		"oldproduid":         "PM2000",
		"oldchannelserialno": p.Model.OutTradeNo,
		"oldmerchantorderid": p.Model.OutTradeNo,
	}

	reqDataJson, _ := json.Marshal(reqData)

	floger.Debug5("des 加密前的数据reqDataJson:", string(reqDataJson))

	reqDataDes, _ := EncryptDES(Key, string(reqDataJson))

	floger.Debug5("des 加密后的数据reqDataDes:", reqDataDes)

	signContent := reqDataDes + "&key=" + Md5Key

	floger.Debug5("md5 签名前 signContent:", signContent)

	signMsg := strings.ToLower(helper.Md5(signContent))

	floger.Debug5("md5 签名后 signMsg:", signMsg)

	headMap := map[string]interface{}{
		"version":         "V1.0",
		"charset":         "1",
		"platid":          PlatId,
		"produid":         "PM4000",
		"channelserialno": p.Model.OutTradeNo,
		"channeltime":     time.Now().Format("20060102150405"),
		"reqdata":         reqDataDes,
		"sign":            signMsg,
	}

	sendJson, _ := json.Marshal(headMap)

	floger.Debug5("发送数据 sendJson:", string(sendJson))
	resp, err := doRequest(nativePayRequestUrl, http.MethodPost, sendJson)
	if err != nil {
		floger.Error("Failed to request pmManagerOrderQuery, err:", err)
		return res, err
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		floger.Error("Failed to json_decode response:", err)
		return res, err
	}
	floger.Debug5("返回结果 resultStr:", res)
	// 解密
	if res.ReturnCode != "0000" {
		return res, errors.New(res.ReturnMsg)
	}
	res.RespData, _ = DecryptDES(Key, res.RespData)
	if err = json.Unmarshal([]byte(res.RespData), &res.ResData); err != nil {
		floger.Error("Failed to json_decode response:", err)
		return res, err
	}
	return res, nil
}

func Notify() {

}

// 发送请求
func doRequest(uri, method string, data interface{}) (*http.Response, error) {
	client := &http.Client{
		// 忽略证书验证
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	var httpReq *http.Request
	baseURL := uri
	if method == http.MethodPost {
		httpReq, _ = http.NewRequest(method, baseURL, bytes.NewBuffer(data.([]byte)))
	} else {
		httpReq, _ = http.NewRequest(method, baseURL, nil)
		params := data.(url.Values)
		httpReq.URL.RawQuery = params.Encode()
		floger.Debug5("Request URL:", httpReq.URL.String())
	}

	// Set the content type to application/type
	httpReq.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	return resp, nil
}

func GenerateChannelSerialNumber() string {
	// Generate 16 random bytes
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return ""
	}

	// Encode bytes to hexadecimal string
	randomString := hex.EncodeToString(randomBytes)
	return randomString
}
