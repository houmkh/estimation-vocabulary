package vacabulary

import (
	"encoding/json"
	_alg "estimation-vocabulary/algorithm"
	_internal "estimation-vocabulary/internal"
	_model "estimation-vocabulary/internal/model"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"time"
)

// 可选等级显示结构
type LevelStruct struct {
	Label string `json:"lable"`
	Value string `json:"value"`
}

// 可选等级列表
var levels = []LevelStruct{
	// TODO 创建一个级别的结构返回,大概三个吧，根据业务看看怎么设置
	// 例子 {Label:"小白",Value:"A1"}
	{Label: "小白", Value: "A1"},
}

// 用于存储每个等级的上下限
var levelVocabulary = map[string][2]int64{
	"A1": {500, 1000},
	"A2": {1000, 2000},
	"B1": {2000, 3000},
	"B2": {3000, 5000},
	"C1": {5000, 8000},
	"C2": {8000, 10000},
}

// TODO 批处理接收文件格式 - 暂定这种,具体还得取决于批处理的具体含义
/*
	{
		"wordList":{
			{"word":a,"known":false},
			{"word":b,"known":true},
		}
	}
*/

// 批处理接收结构
type Batch struct {
	WordList []VocabularyBatch `json:"wordList"`
}

type VocabularyBatch struct {
	Word  string `json:"word"`
	Known bool   `json:"known"`
}

// 选中单词认识与否的请求结构
type WordKnownReq struct {
	TestId string `json:"test_id"`
	WordId string `json:"word_id"`
	Word   string `json:"word"`
	Known  bool   `json:"known"`
}

// 获取单词的请求结构 GetWord（）
type WordGetReq struct {
	TestId string `json:"test_id"`
	//TotalNum int64  `json:"total_num"`
}

// 获取单词的响应结构
type WordGetRes struct {
	TestId   string `json:"test_id"`
	WordId   string `json:"word_id"`
	Word     string `json:"word"`
	TotalNum int64  `json:"total_num"`
}

// @Method GET
// Describe 显示可选的初始级别
func ShowLevelList(c *gin.Context) {
	_internal.ResponseSuccess(c, levels)
}

// @Method Get
// @Param level string
func StartTest(c *gin.Context) {
	// TODO 从前端拿到用户初始所选level
	level := "A1"

	//
	//level := c.Query("level")
	score := levelVocabulary[level][0]
	//fmt.Println(level)

	//TODO 根据level设置初始的Score（Score初始值为当前等级的区间下限）

	// 2. 创建一个map
	testId := uuid.New().String()
	_internal.UserMap.Store(testId, &_internal.UserTestStruct{
		Level:    level,
		Score:    score,
		TotalNum: 0,
		//VocabularyInfo: &_alg.VocabularyInfo{},
		VocabularyInfo: &_alg.VocabularyInfo{},
		LadderInfo:     make(map[string]*_alg.LadderInfo),
		WordInfo:       make(map[string][]int64),
		EndFlag:        false,
		StartTime:      time.Now(),
	})

	// 3.返回一个testId
	_internal.ResponseSuccess(c, testId)
}

//	@Method Get
//  @Param test_id
//  @Return word,total_num,

func GetWord(c *gin.Context) {
	// 1. TODO 接收testId,只有一个方法就直接从路由里读取算了

	testId := uuid.New().String()
	//var totalNum int64 = 0
	testId = c.Query("test_id")
	//totalNum = c.Query("total_num")

	// 2. 判断是否存在对应的map，没有直接给他停了
	userInfo, exist := _internal.UserMap.Load(testId)
	if !exist {
		_internal.ResponseError(c, _internal.CodeInvalidTestId)
		return
	}
	user := userInfo.(*_internal.UserTestStruct)

	// 3.TODO model那边根据level随机拿一个，且不重复
	v := _model.Vocabulary{
		Level: user.Level,
	}

	// 抽取单词且保证单词不重复
	for {
		err := v.SelectVocabularyByLevelRandom()
		if err != nil {
			_internal.ResponseError(c, _internal.CodeWordSelectErr)
		}

		//判断随机抽出来的单词是否重复
		//TODO 解决uuid存储int问题（uuid是int128）
		ok, err := _internal.JudgeIfRepeated(testId, v.Level, v.Id)
		if err != nil {
			_internal.ResponseError(c, _internal.CodeWordRepeat)
		}
		if ok {
			break
		}
	}
	fmt.Println(v, "&&&&&&&&&&&&&&")

	//修改user信息
	user.TotalNum++
	user.VocabularyInfo = &_alg.VocabularyInfo{
		WordId: v.Id,
		Word:   v.Word,
		Known:  false, //先初始化为false
	}
	// 4.TODO 返回需要的数据
	res := WordGetRes{
		TestId:   testId,
		WordId:   strconv.FormatInt(v.Id, 10),
		Word:     v.Word,
		TotalNum: user.TotalNum,
	}

	//返回
	_internal.ResponseSuccess(c, res)
}

// @Description 接收对于单词的认识与否，调用预测算法，重新计算Level
// @Method POST

func UpdateLevel(c *gin.Context) {
	//获取用户每个单词的认识与否,并告知算法
	/*
		testid:xx,
		wordId:1,
		Known:true/false
	*/

	// TODO 1.获得请求参数
	testId := uuid.New().String()
	// body 里的 json 解析方法
	wordReq := WordKnownReq{}
	if err := c.ShouldBindJSON(&wordReq); err != nil {
		log.Println("解析body错误", err)
		_internal.ResponseError(c, _internal.CodeErrParseBody)
		return
	}
	testId = wordReq.TestId
	fmt.Println(wordReq, "&&&&&&&&&&")
	/*
		调用算法
		wordId:1,
		curNum:2,
		curKnown:0,
		Known:true/false,
		Score:3000,
	*/
	// TODO 2.从全局map获取当前testId的一些数据，然后构造算法需要的结构
	userTestMap, exist := _internal.UserMap.Load(testId)
	if !exist {
		_internal.ResponseError(c, _internal.CodeInvalidTestId)
		return
	}

	user := userTestMap.(*_internal.UserTestStruct)
	fmt.Println(user, "*************")
	//TODO 更新userTestStruct的LadderInfo 信息(算法里面实现了这个逻辑，这段去掉)
	//level := user.Level
	//
	////初始化当前等级的LaderInfo
	//if _, ok := user.LadderInfo[level]; !ok {
	//	user.LadderInfo[level] = &_alg.LadderInfo{
	//		CurNum:   0,
	//		KnownNun: 0,
	//	}
	//}
	//user.LadderInfo[level].CurNum++
	//if wordReq.Known {
	//	user.LadderInfo[level].KnownNun++
	//}

	//TODO 更新userTestStruct的VocabularyInfo
	//wordId 将string转int64
	wordIdInt, err := strconv.ParseInt(wordReq.WordId, 10, 64)
	if err != nil {
		_internal.ResponseError(c, _internal.CodeErrParseInt)
	}
	user.VocabularyInfo = &_alg.VocabularyInfo{
		WordId: wordIdInt,
		Word:   wordReq.Word,
		Known:  wordReq.Known,
	}

	// TODO 构造算法需要的结构，具体根据算法需求改.

	userInfo := &_alg.UserInfo{
		Score:          user.Score,
		TotalNum:       user.TotalNum,
		LadderInfo:     user.LadderInfo,
		VocabularyInfo: user.VocabularyInfo,
		EndFlag:        user.EndFlag,
		Level:          user.Level,
	}
	fmt.Println(userInfo)

	// TODO 3.调用算法层，参数统一为UserInfo结构,具体怎么调用看算法层的方法,然后根据返回结构去修改全局map的信息
	// ladderInfo,exist := _internal.UserMap[testId]
	//调用算法层
	fmt.Println("))))))))))))))))")
	_alg.LadderHandler(userInfo)
	fmt.Println("$$$$$$$$$$$$$$")
	fmt.Println(userInfo, "@@@@@@@@@@@@@")
	//覆盖算法返回结果
	user.Score = userInfo.Score
	user.TotalNum = userInfo.TotalNum
	user.LadderInfo = userInfo.LadderInfo
	user.Level = userInfo.Level

	// TODO 4.返回前端，告知请求成功，正常的话不需要数据返回

	_internal.ResponseSuccess(c, nil)
	/*
		return
		score:
		level:
	*/
}

// 接口
func GetResult(c *gin.Context) {
	// 返回结果
	// 1. TODO 获得testid
	testId := c.Query("test_id")

	userTestMap, exist := _internal.UserMap.Load(testId)
	if !exist {
		_internal.ResponseError(c, _internal.CodeInvalidTestId)
	}
	user := userTestMap.(*_internal.UserTestStruct)

	// 2. TODO 调用forcastVocabulary（）
	userInfo := &_alg.UserInfo{
		Score:          user.Score,
		TotalNum:       user.TotalNum,
		LadderInfo:     user.LadderInfo,
		VocabularyInfo: user.VocabularyInfo,
		EndFlag:        user.EndFlag,
		Level:          user.Level,
	}

	_alg.ForecastVocabulary(userInfo)
	//这里省略赋值回userTestStruct,直接返回
	user.Score = userInfo.Score
	user.TotalNum = userInfo.TotalNum
	user.LadderInfo = userInfo.LadderInfo
	user.Level = userInfo.Level

	score := user.Score
	_internal.ResponseSuccess(c, score)
}

func Exit(c *gin.Context) {
	// 1.TODO 获取testid
	testId := uuid.New().String()
	testId = c.Query("test_id")
	// 2.TODO 删掉map
	_internal.UserMap.Delete(testId)
	// 3. TODO 返回前端是否成功
	_internal.ResponseSuccess(c, nil)
}

func Test(c *gin.Context) {
	fmt.Println("test service success")
	_internal.ResponseSuccess(c, nil)
}

// @Method POST
// @Parm form-data
// @Describe 批处理
func GetScoreBatch(c *gin.Context) {
	file, _ := c.FormFile("file")
	// 识别后缀，这里直接限制json
	ext := filepath.Ext(file.Filename)
	if ext != ".json" {
		log.Println("文件类型出错,批处理需要json文件")
		_internal.ResponseError(c, _internal.CodeErrFileFormat)
		return
	}

	src, _ := file.Open()
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		log.Println("读取文件数据错误", err)
		_internal.ResponseError(c, _internal.CodeServerBusy)
		return
	}

	var batchData Batch
	err = json.Unmarshal(data, &batchData)
	if err != nil {
		log.Println("解析json错误", err)
		_internal.ResponseError(c, _internal.CodeErrJsonFormat)
		return
	}

	// TODO 如何根据解析出的json去调用我们自己的方法

	// TODO 计算出最后成绩然后返回

	//_internal.ResponseSuccess()
}

//TODO 逻辑
//TODO 1、建立测试链接：创建testId，加入map，返回testId

//TODO 2、前端接收到链接建立成功的状态码，则发起请求获取单词（需要携带当前的总题目数）

//TODO 3、获取单词之后，前端发起单词认识辨别的请求，后端调用算法对该用户的level进行更新
//TODO 4、前端受到level更新成功的返回后，继续调用获取单词的请求
//TODO 5、当前端获取单词到达某一数目的时候(该计数器由前端和后端一同保持),前端可以选择发起结束请求
//TODO 6、后端处理结束请求，调用算法的forecastVocabulary函数，然后将Score返回给前端展示
