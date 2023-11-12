package database

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	gorp "gopkg.in/gorp.v1"
)

var (
	stats         = make(map[int]*Stat)
	stMutex       sync.RWMutex
	startingStats = map[int]*Stat{
		50: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1017}, // BEAST MALE
		51: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1017}, // BEAST FEMALE
		52: {STR: 13, DEX: 12, INT: 8, HP: 72, MaxHP: 72, CHI: 48, MaxCHI: 48, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1015}, // Monk
		53: {STR: 12, DEX: 12, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1012}, // Male Blade
		54: {STR: 11, DEX: 13, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1012}, // Female Blade
		56: {STR: 15, DEX: 10, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Axe
		57: {STR: 14, DEX: 11, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Female Spear
		59: {STR: 10, DEX: 18, INT: 2, HP: 84, MaxHP: 84, CHI: 36, MaxCHI: 36, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1013}, // Dual Sword

		60: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1017}, // BEAST MALE
		61: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1017}, // BEAST FEMALE
		62: {STR: 13, DEX: 12, INT: 8, HP: 72, MaxHP: 72, CHI: 48, MaxCHI: 48, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1015}, // Divine Monk
		63: {STR: 12, DEX: 12, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1012}, // Divine Male Blade
		64: {STR: 11, DEX: 13, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1012}, // Divine Female Blade
		66: {STR: 15, DEX: 10, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Divine Axe
		67: {STR: 14, DEX: 11, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Divine Female Spear
		69: {STR: 10, DEX: 18, INT: 2, HP: 84, MaxHP: 84, CHI: 36, MaxCHI: 36, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1013}, // Divine Dual Sword

		70: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1017}, // BEAST MALE
		71: {STR: 10, DEX: 18, INT: 2, HP: 66, MaxHP: 66, CHI: 18, MaxCHI: 18, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1017}, // BEAST FEMALE
		72: {STR: 13, DEX: 12, INT: 8, HP: 72, MaxHP: 72, CHI: 48, MaxCHI: 48, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1015}, // Dark Monk
		73: {STR: 12, DEX: 12, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1012}, // Dark Male Blade
		74: {STR: 11, DEX: 13, INT: 6, HP: 90, MaxHP: 90, CHI: 30, MaxCHI: 30, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1002}, // Dark Female Blade
		76: {STR: 15, DEX: 10, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dark Axe
		77: {STR: 14, DEX: 11, INT: 5, HP: 96, MaxHP: 96, CHI: 24, MaxCHI: 24, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1000}, // Dark Female Spear
		79: {STR: 10, DEX: 18, INT: 2, HP: 84, MaxHP: 84, CHI: 36, MaxCHI: 36, HPRecoveryRate: 10, CHIRecoveryRate: 10, AttackSpeed: 1013}, // Dark Dual Sword
	}
)

type Stat struct {
	ID           int `db:"id"`
	STR          int `db:"str"`
	DEX          int `db:"dex"`
	INT          int `db:"int"`
	StatPoints   int `db:"stat_points"`
	Honor        int `db:"honor"`
	Wind         int `db:"wind"`
	Water        int `db:"water"`
	Fire         int `db:"fire"`
	NaturePoints int `db:"nature_points"`
	HP           int `db:"hp"`
	CHI          int `db:"chi"`

	AttackSpeed           int `db:"-"`
	AdditionalAttackSpeed int `db:"-"`

	MaxHP          int `db:"-"`
	HPRecoveryRate int `db:"-"`

	MaxCHI          int `db:"-"`
	CHIRecoveryRate int `db:"-"`

	STRBuff int `db:"-"`
	DEXBuff int `db:"-"`
	INTBuff int `db:"-"`

	MinATK       int `db:"-"`
	MaxATK       int `db:"-"`
	ATKRate      int `db:"-"`
	MinArtsATK   int `db:"-"`
	MaxArtsATK   int `db:"-"`
	ArtsATKRate  int `db:"-"`
	DEF          int `db:"-"`
	DefRate      int `db:"-"`
	ArtsDEF      int `db:"-"`
	ArtsDEFRate  int `db:"-"`
	Accuracy     int `db:"-"`
	Dodge        int `db:"-"`
	PoisonATK    int `db:"-"`
	ParalysisATK int `db:"-"`
	ConfusionATK int `db:"-"`
	PoisonDEF    int `db:"-"`
	ParalysisDEF int `db:"-"`
	ConfusionDEF int `db:"-"`

	WindBuff int `db:"-"`

	WaterBuff int `db:"-"`

	FireBuff int `db:"-"`

	Paratime      int `db:"-"`
	PoisonTime    int `db:"-"`
	ConfusionTime int `db:"-"`

	PVPdef                int     `db:"-"`
	AdditionalPVPdefRate  float32 `db:"-"`
	AdditionalPVPsdefRate float32 `db:"-"`
	PVPsdef               int     `db:"-"`
	PVPdmg                int     `db:"-"`
	PVPsdmg               int     `db:"-"`

	DEXDamageReduction float32 `db:"-"`
	ShopMultiplier     float32 `db:"-"`

	CriticalRate                   int `db:"-"`
	CriticalProbability            int `db:"-"`
	DamageReflectedRate            int `db:"-"`
	DamageReflectedProbabilty      int `db:"-"`
	DamagedAbsobedRate             int `db:"-"`
	DamagedAbsorbedProbabilty      int `db:"-"`
	DamageConvertedToHpRate        int `db:"-"`
	DamageConvertedToHpProbability int `db:"-"`
	IncreasedPVPDamageRate         int `db:"-"`
	IncreasedPVPDamageProbablility int `db:"-"`

	ExpMultiplier    float64 `db:"-"`
	DropMultiplier   float64 `db:"-"`
	GoldMultiplier   float64 `db:"-"`
	PetExpMultiplier float64 `db:"-"`

	AdditionalSkillRadius float64 `db:"-"`

	EnhancedProbabilitiesBuff int `db:"-"`
	SyntheticCompositeBuff    int `db:"-"`
	AdvancedCompositeBuff     int `db:"-"`
	HyeolgongCost             int `db:"-"`

	AdditionalRunningSpeed float64 `db:"-"`

	Npc_gold_multiplier float64 `db:"-"`
}

func (t *Stat) Create(c *Character) error {
	t = startingStats[c.Type]
	t.ID = c.ID
	t.StatPoints = 4
	t.NaturePoints = 0
	t.Honor = 0
	return pgsql_DbMap.Insert(t)
}

func (t *Stat) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(t)
}

func (t *Stat) Update() error {
	_, err := pgsql_DbMap.Update(t)
	return err
}

func (t *Stat) Delete() error {
	stMutex.Lock()
	delete(stats, t.ID)
	stMutex.Unlock()

	_, err := pgsql_DbMap.Delete(t)
	return err
}

func (t *Stat) Calculate() error {

	c, err := FindCharacterByID(t.ID)
	if err != nil {
		return err
	} else if c == nil {
		return nil
	}

	temp := *t

	stStat := startingStats[c.Type]

	temp.MaxHP = stStat.MaxHP
	temp.MaxCHI = stStat.MaxCHI

	if temp.HP <= 0 {
		temp.HP = int(float32(temp.MaxHP) * 0.2)
	}

	temp.STRBuff = 0
	temp.DEXBuff = 0
	temp.INTBuff = 0
	temp.WindBuff = 0
	temp.WaterBuff = 0
	temp.FireBuff = 0
	temp.MinATK = temp.STR
	temp.MaxATK = temp.STR
	temp.ATKRate = 0
	temp.MinArtsATK = temp.STR
	temp.MaxArtsATK = temp.STR
	temp.ArtsATKRate = 0
	temp.DEF = temp.DEX
	temp.DefRate = 0
	temp.ArtsDEF = 2*temp.INT + temp.DEX
	temp.ArtsDEFRate = 0
	temp.Accuracy = int(float32(temp.STR) * 0.925)
	temp.Dodge = temp.DEX
	temp.PoisonATK = 0
	temp.PoisonDEF = 0
	temp.ParalysisATK = 0
	temp.ParalysisDEF = 0
	temp.ConfusionATK = 0
	temp.ConfusionDEF = 0
	temp.PoisonTime = 0
	temp.Paratime = 0
	temp.ConfusionTime = 0
	temp.CHIRecoveryRate = 10
	temp.HPRecoveryRate = 10
	temp.DEXDamageReduction = 0
	temp.ShopMultiplier = 0
	temp.AdditionalPVPdefRate = 0
	temp.PVPdef = 0
	temp.PVPdmg = 0
	temp.AdditionalPVPsdefRate = 0
	temp.PVPsdef = 0
	temp.PVPsdmg = 0
	temp.ExpMultiplier = 1
	temp.DropMultiplier = 1
	temp.GoldMultiplier = 1
	temp.AdditionalSkillRadius = 0
	temp.AttackSpeed = stStat.AttackSpeed + temp.DEX
	temp.EnhancedProbabilitiesBuff = 0
	temp.SyntheticCompositeBuff = 0
	temp.AdvancedCompositeBuff = 0
	temp.HyeolgongCost = 0
	temp.Npc_gold_multiplier = 1

	temp.CriticalRate = 0
	temp.CriticalProbability = 5
	temp.DamageReflectedProbabilty = 0
	temp.DamagedAbsorbedProbabilty = 0
	temp.DamageConvertedToHpProbability = 0
	temp.DamageReflectedRate = 0
	temp.DamageConvertedToHpRate = 0
	temp.DamagedAbsobedRate = 0
	temp.AdditionalRunningSpeed = 0
	temp.PetExpMultiplier = 0

	c.BuffEffects(&temp)
	c.JobPassives(&temp)

	c.AntiExp = false
	c.CanMove = true

	c.ItemEffects(&temp, 0, 9)         // NORMAL ITEMS
	c.ItemEffects(&temp, 307, 315)     // HT ITEMS
	c.ItemEffects(&temp, 0x0B, 0x43)   // INV BUFFS1
	c.ItemEffects(&temp, 0x155, 0x18D) // INV BUFFS2
	c.ItemEffects(&temp, 397, 401)     // MARBLES(1-5)

	totalDEX := temp.DEX + temp.DEXBuff
	totalWind := temp.Wind + temp.WindBuff
	totalWater := temp.Water + temp.WaterBuff
	totalFire := temp.Fire + temp.FireBuff

	temp.CriticalRate += totalDEX / 20

	if totalDEX <= 300 {
		temp.DEXDamageReduction += float32(totalDEX) * 0.14
	} else if totalDEX > 300 && totalDEX <= 600 {
		temp.DEXDamageReduction += 300 * 0.14
		temp.DEXDamageReduction += (float32(totalDEX) - 300) * 0.07
	} else if totalDEX > 600 {
		temp.DEXDamageReduction += 300 * 0.14
		temp.DEXDamageReduction += 300 * 0.07
		temp.DEXDamageReduction += (float32(totalDEX) - 600) * 0.04
	}

	temp.DEF += temp.DEXBuff + 2*totalWind + 1*totalWater + 1*totalFire
	temp.DEF += temp.DEF * temp.DefRate / 100
	temp.ArtsDEF += 2*temp.INTBuff + temp.DEXBuff + 1*totalWind + 2*totalWater + 1*totalFire
	temp.ArtsDEF += temp.ArtsDEF * temp.ArtsDEFRate / 100

	totalSTR := temp.STR + temp.STRBuff
	totalINT := temp.INT + temp.INTBuff

	temp.MaxHP += 10 * totalSTR
	temp.MaxCHI += 3 * totalINT

	temp.MinATK += temp.STRBuff + 1*totalWind + 1*totalWater + 2*totalFire
	temp.MinATK += temp.MinATK * temp.ATKRate / 100
	temp.MaxATK += temp.STRBuff + 1*totalWind + 1*totalWater + 2*totalFire
	temp.MaxATK += temp.MaxATK * temp.ATKRate / 100

	temp.MinArtsATK += temp.STRBuff + 2*totalINT + int(float32(totalINT*temp.MinATK)/200)
	temp.MinArtsATK += temp.MinArtsATK * temp.ArtsATKRate / 100
	temp.MaxArtsATK += temp.STRBuff + 2*totalINT + int(float32(totalINT*temp.MaxATK)/200)
	temp.MaxArtsATK += temp.MaxArtsATK * temp.ArtsATKRate / 100

	temp.Accuracy += int(float32(temp.STRBuff) * 0.925)
	temp.Dodge += temp.DEXBuff

	temp.PVPdef = temp.DEF + int((float32(temp.DEF) * temp.AdditionalPVPdefRate))
	temp.PVPsdef = temp.ArtsDEF + int((float32(temp.ArtsDEF) * temp.AdditionalPVPsdefRate))

	if c.Injury > 70 {
		injury := float32(0.7)
		if c.Injury > 80 {
			injury = float32(0.5)
		}
		if c.Injury > 90 {
			injury = float32(0.3)
		}
		temp.MinATK = int(float32(temp.MinATK) * injury)
		temp.MaxATK = int(float32(temp.MaxATK) * injury)
		temp.MinArtsATK = int(float32(temp.MinArtsATK) * injury)
		temp.MaxArtsATK = int(float32(temp.MaxArtsATK) * injury)

		temp.Accuracy = int(float32(temp.Accuracy) * injury)
		temp.DEF = int(float32(temp.DEF) * injury)
		temp.ArtsDEF = int(float32(temp.ArtsDEF) * injury)
		temp.DefRate = int(float32(temp.DefRate) * injury)
	}

	temp.AttackSpeed += t.AdditionalAttackSpeed
	slots, err := c.InventorySlots()
	if err == nil && slots[c.WeaponSlot].ItemID != 0 {
		item, ok := GetItemInfo(slots[c.WeaponSlot].ItemID)
		if !ok {
			return errors.New("Item not found")
		}
		attackspeed := 0
		if item.Type == 102 || item.Type == 105 || item.Type == 71 || item.Type == 101 {
			attackspeed = 10
		} else if item.Type == 107 {
			attackspeed = 4
		} else if item.Type == 100 || item.Type == 108 {
			attackspeed = 12
		} else if item.Type == 103 {
			attackspeed = 6
		} else {
			attackspeed = 8
		}
		temp.AttackSpeed += attackspeed
	}
	if temp.AttackSpeed > 2000 {
		temp.AttackSpeed = 2000
	}

	*t = temp
	go t.Update()
	return nil
}

func (t *Stat) Reset() error {

	c, err := FindCharacterByID(t.ID)
	if err != nil {
		return err
	} else if c == nil {
		return fmt.Errorf("CalculateTotalStatPoints: character not found")
	}

	tens := c.Level / 10
	statPts := ((((tens + 3) * (tens + 4) / 2) - 6) * 10) - 4
	statPts += (tens + 4) * ((c.Level + 1) % 10)
	if c.Level < 101 {
		statPts += c.Level * int(c.Reborns)
	}

	if c.Level > 100 {
		np := (c.Level - 100) * 4
		t.NaturePoints = np
		statPts += 100 * int(c.Reborns)
	}

	stat := startingStats[c.Type]
	t.STR = stat.STR
	t.DEX = stat.DEX
	t.INT = stat.INT
	t.Wind = 0
	t.Fire = 0
	t.Water = 0

	t.StatPoints = statPts

	return t.Update()
}

func FindStatByID(id int) (*Stat, error) {

	stMutex.RLock()
	s, ok := stats[id]
	stMutex.RUnlock()
	if ok {
		return s, nil
	}

	stat := &Stat{}
	query := `select * from hops.stats where id = $1`

	if err := pgsql_DbMap.SelectOne(&stat, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindStatByID: %s", err.Error())
	}

	stMutex.Lock()
	defer stMutex.Unlock()
	stats[stat.ID] = stat

	return stat, nil
}

func GetAllStats() (map[int]*Stat, error) {

	var arr []*Stat
	query := `select * from hops.stats`

	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("Rank: %s", err.Error())
	}

	stats := make(map[int]*Stat)
	for _, npc := range arr {
		stats[npc.ID] = npc
	}

	return stats, nil
}
func DeleteStatFromCache(id int) {
	delete(stats, id)
}
func DeleteUnusedStats() {
	stats, err := GetAllStats()
	if err != nil {
		return
	}
	for _, stat := range stats {
		char, err := FindCharacterByID(stat.ID)
		if err != nil {
			continue
		}
		if char == nil {
			stat.Delete()
		}
	}
}
func (t *Stat) CalculateHonorIDs() error {
	if t == nil {
		return nil
	}
	c, err := FindCharacterByID(t.ID)
	if err != nil {
		return err
	} else if c == nil {
		return nil
	}
	if t.Honor >= 30000 {
		c.HonorRank = 1
	} else if t.Honor >= 20000 {
		c.HonorRank = 2
	} else if t.Honor >= 10000 {
		c.HonorRank = 4
	} else if t.Honor >= 5000 {
		c.HonorRank = 14
	} else if t.Honor >= 5000 {
		c.HonorRank = 30
	} else if t.Honor >= 1000 {
		c.HonorRank = 50
	} else {
		c.HonorRank = 0
	}
	return nil
}
