package model

import (
	"math/rand"
	"time"
)

// vocabularies
type Vocabulary struct {
	Id             int64     `json:"id" gorm:"column:id"`
	Word           string    `json:"word" gorm:"column:word"`
	Level          string    `json:"level" gorm:"column:level"`
	FrequenceLevel int       `json:"frequence_level" gorm:"column:frequence_level"`
	CreatedAt      time.Time `json:"createAt" `
	UpdatedAt      time.Time `json:"updateAt"`
	DeleteFlag     int       `json:"delete_flag" gorm:"column:delete_flag"`
}

func (v *Vocabulary) InsertVocabulary() (err error) {
	v.DeleteFlag = 0
	result := db.Model(&v).Create(&v)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

//func (v *Vocabulary) SelectVocabularyByLevelRandom() error {
//	// 需要保证不重复
//
//}

func (v *Vocabulary) SelectVocabularyByLevelRandom() error {
	// 在业务逻辑层保证抽取的单词不重复，这里只负责随机抽取
	//利用Gorm设置随机数种子进行随机抽取

	//设置随机数种子
	rand.Seed(time.Now().UnixNano())

	err := db.Model(&Vocabulary{}).Where("level =?", v.Level).
		Order("RAND()").
		Limit(1).
		Find(&v).
		Error
	if err != nil {
		return err
	}
	return nil
}

//func (v *Vocabulary) SelectByID() error {
//	result := db.Model(&Vocabulary{}).Where("id=?", id).Select(&v)
//
//	if result.Error != nil {
//		return result.Error
//	}
//}
