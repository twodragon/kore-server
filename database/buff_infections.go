package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
	gorp "gopkg.in/gorp.v1"
)

var (
	BuffInfections = make(map[int]*BuffInfection)
)

type BuffInfection struct {
	ID                                int    `db:"id"`
	Name                              string `db:"name"`
	Description                       string
	Icon                              string
	PoisonDef                         int     `db:"poison_def"`
	AdditionalPoisonDef               int     `db:"additional_poison_def"`
	ParalysisDef                      int     `db:"paralysis_def"`
	AdditionalParalysisDef            int     `db:"additional_para_def"`
	ConfusionDef                      int     `db:"confusion_def"`
	AdditionalConfusionDef            int     `db:"additional_confusion_def"`
	BaseDef                           int     `db:"base_def"`
	AdditionalDEF                     int     `db:"additional_def"`
	ArtsDEF                           int     `db:"arts_def"`
	AdditionalArtsDEF                 int     `db:"additional_arts_def"`
	MaxHP                             int     `db:"max_hp"`
	HPRecoveryRate                    int     `db:"hp_recovery_rate"`
	STR                               int     `db:"str"`
	AdditionalSTR                     int     `db:"additional_str"`
	DEX                               int     `db:"dex"`
	AdditionalDEX                     int     `db:"additional_dex"`
	INT                               int     `db:"int"`
	AdditionalINT                     int     `db:"additional_int"`
	Wind                              int     `db:"wind"`
	AdditionalWind                    int     `db:"additional_wind"`
	Water                             int     `db:"water"`
	AdditionalWater                   int     `db:"additional_water"`
	Fire                              int     `db:"fire"`
	AdditionalFire                    int     `db:"additional_fire"`
	AdditionalHP                      int     `db:"additional_hp"`
	BaseATK                           int     `db:"base_atk"`
	AdditionalATK                     int     `db:"additional_atk"`
	BaseArtsATK                       int     `db:"base_arts_atk"`
	AdditionalArtsATK                 int     `db:"additional_arts_atk"`
	Accuracy                          int     `db:"accuracy"`
	AdditionalAccuracy                int     `db:"additional_accuracy"`
	DodgeRate                         int     `db:"dodge_rate"`
	AdditionalDodgeRate               int     `db:"additional_dodge_rate"`
	MovSpeed                          float64 `db:"movement_speed"`
	AdditionalMovSpeed                float64 `db:"additional_movement_speed"`
	ExpRate                           int     `db:"exp_rate"`
	DropRate                          int     `db:"drop_rate"`
	HyeolgongCost                     int     `db:"hyeolgong_cost"`
	NPCSellingCost                    int     `db:"npc_selling"`
	NPCBuyingCost                     int     `db:"npc_buying"`
	IsPercent                         int     `db:"ispercent"`
	LightningRadius                   int     `db:"lightning_radius"`
	AttackSpeed                       int     `db:"attack_speed"`
	AdditionalHPRecovery              int     `db:"additional_hp_recovery"`
	MaxChi                            int     `db:"max_chi"`
	DamageReflection                  int     `db:"damage_reflection"`
	DropItem                          int     `db:"drop_item"`
	EnchancedProb                     int     `db:"enchanced_prob"`
	SyntheticComposite                int     `db:"synthetic_composite"`
	AdvancedComposite                 int     `db:"advanced_composite"`
	PetMaxHP                          int     `db:"pet_base_hp"`
	PetAdditionalHP                   int     `db:"pet_additional_hp"`
	PetBaseDEF                        int     `db:"pet_base_def"`
	PetAdditionalDEF                  int     `db:"pet_additional_def"`
	PetArtsDEF                        int     `db:"pet_base_arts_def"`
	PetAdditinalArtsDEF               int     `db:"pet_additional_arts_def"`
	AdditionalAttackSpeed             int     `db:"additional_attack_speed"`
	MakeSize                          float64 `db:"makesize"`
	CriticalRate                      int     `db:"critical_strike"`
	AdditionalCritRate                int     `db:"additional_critical_strike"`
	CashAcquired                      int     `db:"cash_acquired"`
	TakingEffectProbability           int     `db:"taking_effect_probability"`
	AdditionalTakingEffectProbability int     `db:"additional_taking_effect_probability"`
	AdditionalDamageReflection        int     `db:"additional_damage_reflection"`
	MaxArtsAtk                        int     `db:"max_arts_atk" json:"max_arts_atk"`
	AdditionalMaxArtsAtk              int
	MinArtsAtk                        int `db:"min_arts_atk" json:"min_arts_atk"`
	AdditionalMinArtATK               int

	PetExpMultiplier float64 `db:"pet_exp_rate" json:"pet_exp_rate"`
}

func (e *BuffInfection) Create() error {
	return pgsql_DbMap.Insert(e)
}

func (e *BuffInfection) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *BuffInfection) Delete() error {
	_, err := pgsql_DbMap.Delete(e)
	return err
}

func (e *BuffInfection) Update() error {
	_, err := pgsql_DbMap.Update(e)
	return err
}

func GetBuffInfections() error {
	log.Print("Reading Save Points...")
	f, err := excelize.OpenFile("data/tb_buff_infection.xlsx")
	if err != nil {
		return err
	}

	defer f.Close()
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return err
	}
	for index, row := range rows {
		if index == 0 {
			continue
		}
		BuffInfections[utils.StringToInt(row[1])] = &BuffInfection{
			ID:                     utils.StringToInt(row[1]),
			Name:                   row[2],
			Description:            row[3],
			IsPercent:              utils.StringToInt(row[12]),
			BaseATK:                utils.StringToInt(row[29]),
			AdditionalATK:          utils.StringToInt(row[30]),
			BaseArtsATK:            utils.StringToInt(row[31]),
			AdditionalArtsATK:      utils.StringToInt(row[32]),
			MinArtsAtk:             utils.StringToInt(row[33]),
			AdditionalMinArtATK:    utils.StringToInt(row[34]),
			MaxArtsAtk:             utils.StringToInt(row[35]),
			AdditionalMaxArtsAtk:   utils.StringToInt(row[36]),
			LightningRadius:        utils.StringToInt(row[41]),
			PoisonDef:              utils.StringToInt(row[47]),
			AdditionalPoisonDef:    utils.StringToInt(row[48]),
			ParalysisDef:           utils.StringToInt(row[53]),
			AdditionalParalysisDef: utils.StringToInt(row[53]),
			ConfusionDef:           utils.StringToInt(row[59]),
			AdditionalConfusionDef: utils.StringToInt(row[60]),
			Accuracy:               utils.StringToInt(row[63]),
			AdditionalAccuracy:     utils.StringToInt(row[64]),
			DodgeRate:              utils.StringToInt(row[65]),
			AdditionalDodgeRate:    utils.StringToInt(row[66]),
			BaseDef:                utils.StringToInt(row[67]),
			AdditionalDEF:          utils.StringToInt(row[68]),
			ArtsDEF:                utils.StringToInt(row[69]),
			AdditionalArtsDEF:      utils.StringToInt(row[70]),
			AttackSpeed:            utils.StringToInt(row[71]),
			AdditionalAttackSpeed:  utils.StringToInt(row[72]),
			MovSpeed:               utils.StringToFloat64(row[73]),
			AdditionalMovSpeed:     utils.StringToFloat64(row[74]),
			MaxHP:                  utils.StringToInt(row[75]),
			AdditionalHP:           utils.StringToInt(row[76]),
			MaxChi:                 utils.StringToInt(row[77]),
			//AdditionalChi:          utils.StringToInt(row[78]),
			HPRecoveryRate:                    utils.StringToInt(row[81]),
			AdditionalHPRecovery:              utils.StringToInt(row[82]),
			STR:                               utils.StringToInt(row[89]),
			AdditionalSTR:                     utils.StringToInt(row[90]),
			DEX:                               utils.StringToInt(row[91]),
			AdditionalDEX:                     utils.StringToInt(row[92]),
			INT:                               utils.StringToInt(row[93]),
			AdditionalINT:                     utils.StringToInt(row[94]),
			Wind:                              utils.StringToInt(row[95]),
			AdditionalWind:                    utils.StringToInt(row[96]),
			Water:                             utils.StringToInt(row[97]),
			AdditionalWater:                   utils.StringToInt(row[98]),
			Fire:                              utils.StringToInt(row[99]),
			AdditionalFire:                    utils.StringToInt(row[100]),
			MakeSize:                          utils.StringToFloat64(row[110]),
			CriticalRate:                      utils.StringToInt(row[111]),
			AdditionalCritRate:                utils.StringToInt(row[112]),
			TakingEffectProbability:           utils.StringToInt(row[120]),
			AdditionalTakingEffectProbability: utils.StringToInt(row[121]),
			DamageReflection:                  utils.StringToInt(row[122]),
			AdditionalDamageReflection:        utils.StringToInt(row[123]),
			ExpRate:                           utils.StringToInt(row[128]),
			DropRate:                          utils.StringToInt(row[129]),
			DropItem:                          utils.StringToInt(row[132]),
			EnchancedProb:                     utils.StringToInt(row[136]),
			SyntheticComposite:                utils.StringToInt(row[137]),
			AdvancedComposite:                 utils.StringToInt(row[138]),
			PetExpMultiplier:                  utils.StringToFloat64(row[140]),
			PetMaxHP:                          utils.StringToInt(row[142]),
			PetAdditinalArtsDEF:               utils.StringToInt(row[143]),
			PetBaseDEF:                        utils.StringToInt(row[148]),
			PetAdditionalDEF:                  utils.StringToInt(row[149]),
			PetArtsDEF:                        utils.StringToInt(row[150]),
			//PetAdditionalArtDef : utils.StringToInt(row[151]),
			NPCSellingCost: utils.StringToInt(row[152]),
			NPCBuyingCost:  utils.StringToInt(row[153]),
			HyeolgongCost:  utils.StringToInt(row[154]),
		}
	}

	return nil
}
