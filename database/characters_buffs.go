package database

import (
	"database/sql"
	"fmt"
	"sort"

	gorp "gopkg.in/gorp.v1"
)

type Buff struct {
	ID              int     `db:"id" json:"id"`
	CharacterID     int     `db:"character_id" json:"character_id"`
	Name            string  `db:"name" json:"name"`
	ATK             int     `db:"atk" json:"atk"`
	ATKRate         int     `db:"atk_rate" json:"atk_rate"`
	ArtsATK         int     `db:"arts_atk" json:"arts_atk"`
	ArtsATKRate     int     `db:"arts_atk_rate" json:"arts_atk_rate"`
	PoisonDEF       int     `db:"poison_def" json:"poison_def"`
	ParalysisDEF    int     `db:"paralysis_def" json:"paralysis_def"`
	ConfusionDEF    int     `db:"confusion_def" json:"confusion_def"`
	DEF             int     `db:"def" json:"def"`
	DEFRate         int     `db:"def_rate" json:"def_rate"`
	ArtsDEF         int     `db:"arts_def" json:"arts_def"`
	ArtsDEFRate     int     `db:"arts_def_rate" json:"arts_def_rate"`
	Accuracy        int     `db:"accuracy" json:"accuracy"`
	Dodge           int     `db:"dodge" json:"dodge"`
	MaxHP           int     `db:"max_hp" json:"max_hp"`
	HPRecoveryRate  int     `db:"hp_recovery_rate" json:"hp_recovery_rate"`
	MaxCHI          int     `db:"max_chi" json:"max_chi"`
	CHIRecoveryRate int     `db:"chi_recovery_rate" json:"chi_recovery_rate"`
	STR             int     `db:"str" json:"str"`
	DEX             int     `db:"dex" json:"dex"`
	INT             int     `db:"int" json:"int"`
	EXPMultiplier   int     `db:"exp_multiplier" json:"exp_multiplier"`
	DropMultiplier  int     `db:"drop_multiplier" json:"drop_multiplier"`
	RunningSpeed    float64 `db:"running_speed" json:"running_speed"`
	AttackSpeed     int     `db:"attack_speed" json:"attack_speed"`
	StartedAt       int64   `db:"started_at" json:"started_at"`
	Duration        int64   `db:"duration" json:"duration"`
	BagExpansion    bool    `db:"bag_expansion" json:"bag_expansion"`
	SkillPlus       int     `db:"skill_plus" json:"skill_plus"`
	CanExpire       bool    `db:"can_expire" json:"can_expire"`
	Wind            int     `db:"wind" json:"wind"`
	Water           int     `db:"water" json:"water"`
	Fire            int     `db:"fire" json:"fire"`
	Reflect         int     `db:"reflect" json:"reflect"`
	CriticalRate    int     `db:"critical_strike" json:"critical_strike"`
	GoldMultiplier  float64 `db:"gold_multiplier" json:"gold_multiplier"`
	MaxArtsAtk      int     `db:"max_arts_atk" json:"max_arts_atk"`
	MinArtsAtk      int     `db:"min_arts_atk" json:"min_arts_atk"`
	IsServerEpoch   bool    `db:"server_epoch" json:"server_epoch"`

	EnhancedProbabilitiesBuff int `db:"enchanced_prob" json:"enchanced_prob"`
	SyntheticCompositeBuff    int `db:"synthetic_composite" json:"synthetic_composite"`
	AdvancedCompositeBuff     int `db:"advanced_composite" json:"advanced_composite"`
	HyeolgongCost             int `db:"hyeolgong_cost" json:"hyeolgong_cost"`

	PetExpMultiplier    float64 `db:"pet_exp_rate" json:"pet_exp_rate"`
	Npc_gold_multiplier float64 `db:"npc_gold_multiplier" json:"npc_gold_multiplier"`
}

func (b *Buff) Create() error {
	return pgsql_DbMap.Insert(b)
}

func (b *Buff) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(b)
}

func (b *Buff) Delete() error {
	_, err := pgsql_DbMap.Delete(b)
	return err
}

func (b *Buff) Update() error {
	_, err := pgsql_DbMap.Update(b)
	return err
}

func FindBuffsByCharacterID(characterID int) ([]*Buff, error) {

	var buffs []*Buff
	query := `select * from hops.characters_buffs where character_id = $1`

	if _, err := pgsql_DbMap.Select(&buffs, query, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffsByCharacterID: %s", err.Error())
	}

	sort.Slice(buffs, func(i, j int) bool {
		return buffs[i].StartedAt+buffs[i].Duration <= buffs[j].StartedAt+buffs[j].Duration
	})

	return buffs, nil
}

func FindBuffByID(buffID, characterID int) (*Buff, error) {

	var buff *Buff
	query := `select * from hops.characters_buffs where id = $1 and character_id = $2`

	if err := pgsql_DbMap.SelectOne(&buff, query, buffID, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindBuffByID: %s", err.Error())
	}

	return buff, nil
}
