package player

import (
	"fmt"
	"log"
	"time"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/utils"
)

type (
	AttackHandler        struct{}
	InstantAttackHandler struct{}
	DealDamageHandler    struct{}
	CastSkillHandler     struct{}
	CastMonkSkillHandler struct{}
	RemoveBuffHandler    struct{}
)

var (
	ATTACKED      = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x41, 0x01, 0x0D, 0x02, 0x01, 0x00, 0x00, 0x00, 0x55, 0xAA}
	INST_ATTACKED = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x41, 0x01, 0x0D, 0x02, 0x01, 0x00, 0x00, 0x00, 0x55, 0xAA}

	PVP_DEAL_DAMAGE          = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
	PVP_DEAL_CRITICAL_DAMAGE = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}

	PVP_DEAL_SKILL_DAMAGE          = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x42, 0x75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
	PVP_DEAL_SKILL_CRITICAL_DAMAGE = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x42, 0x75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0xB8, 0x3B, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
)

func (h *AttackHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c.AttackDelay > 0 {
		return nil, nil
	}

	c.AttackMutex.Lock()
	defer c.AttackMutex.Unlock()

	if c == nil {
		return nil, nil
	}

	st := s.Stats
	if st == nil {
		return nil, nil
	}

	aiID := uint16(utils.BytesToInt(data[7:9], true))
	ai, ok := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, aiID).(*database.AI)
	if ok {
		if ai == nil || ai.HP <= 0 || ai.IsDead {
			goto OUT
		}

		npcPos := database.GetNPCPosByID(ai.PosID)
		if npcPos == nil {
			goto OUT
		}

		npc, ok := database.GetNpcInfo(npcPos.NPCID)
		if !ok || npc == nil {
			goto OUT
		}

		if aiID != uint16(c.Selection) {
			goto OUT
		}

		if npcPos.Attackable {
			ai.MovementToken = 0
			ai.IsMoving = false
			ai.TargetPlayerID = c.ID

			dmg, err := c.CalculateDamage(ai, false)
			if err != nil {
				goto OUT
			}

			if diff := int(npc.Level) - c.Level; diff > 0 {
				reqAcc := utils.SigmaFunc(float64(diff))
				if float64(st.Accuracy) < reqAcc {
					probability := float64(st.Accuracy) * 1000 / reqAcc
					if npcPos.NPCID == 50009 || npcPos.NPCID == 50010 {
						probability = 700
					}
					if utils.RandInt(0, 1000) > int64(probability) {
						dmg = 0
					}
				}
			}

			c.Targets = append(c.Targets, &database.Target{Damage: dmg, AI: ai})
		}

	} else if enemy := database.FindCharacter(s.User.ConnectedServer, aiID); enemy != nil {
		enemy := database.FindCharacter(s.User.ConnectedServer, aiID)
		if enemy == nil || !enemy.IsActive {
			goto OUT
		}

		dmg, err := c.CalculateDamageToPlayer(enemy, false)
		if err != nil {
			return nil, err
		}

		c.PlayerTargets = append(c.PlayerTargets, &database.PlayerTarget{Damage: dmg, Enemy: enemy})
	}

OUT:
	resp := ATTACKED
	resp[4] = data[4]
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.IntToBytes(uint64(aiID), 2, true), 9)       // ai id

	p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: resp, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		return nil, err
	}
	if st.PoisonATK > 0 {
		//go ai.Poison(st.PoisonATK, c)
	}
	if st.ParalysisATK > 0 {
		//go ai.Paralysis(st.ParalysisATK)
	}
	if st.ConfusionATK > 0 {
		//go ai.Confusion(st.ConfusionATK)
	}
	c.AttackDelay = 1
	defer time.AfterFunc(time.Second*1, func() {
		c.AttackDelay = 0
	})

	return resp, nil
}

func (h *InstantAttackHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c.AttackDelay > 0 {
		return nil, nil
	}

	c.AttackMutex.Lock()
	defer c.AttackMutex.Unlock()
	if c == nil {
		return nil, nil
	}

	st := s.Stats
	if st == nil {
		return nil, nil
	}

	aiID := uint16(utils.BytesToInt(data[7:9], true))

	ai, ok := database.GetFromRegister(s.User.ConnectedServer, s.Character.Map, aiID).(*database.AI)
	if ok {

		if aiID != uint16(c.Selection) {
			goto OUT
		}

		if ai == nil || ai.HP <= 0 || ai.IsDead {
			goto OUT
		}

		npcPos := database.GetNPCPosByID(ai.PosID)
		if npcPos == nil {
			goto OUT
		}

		npc, ok := database.GetNpcInfo(npcPos.NPCID)
		if !ok || npc == nil {
			goto OUT
		}

		if npcPos.Attackable {
			ai.MovementToken = 0
			ai.IsMoving = false
			ai.TargetPlayerID = c.ID

			dmg := int(utils.RandInt(int64(st.MinATK), int64(st.MaxATK))) - npc.DEF
			if c.Level < 101 {
				dmg = int(utils.RandInt(int64(st.MinATK), int64(st.MaxATK))) - int(npc.Level*int16(c.Reborns)) - npc.DEF
			}
			if dmg < 0 {
				dmg = 0
			} else if dmg > ai.HP {
				dmg = ai.HP
			}

			if diff := int(npc.Level) - c.Level; diff > 0 {
				reqAcc := utils.SigmaFunc(float64(diff))
				if float64(st.Accuracy) < reqAcc {
					probability := float64(st.Accuracy) * 1000 / reqAcc
					if npcPos.NPCID == 50009 || npcPos.NPCID == 50010 {
						probability = 700
					}
					if utils.RandInt(0, 1000) > int64(probability) {
						dmg = 0
					}
				}
			}

			time.AfterFunc(time.Second/2, func() { // attack done
				go c.DealDamage(ai, dmg, false)
			})
		}

	} else if enemy := database.FindCharacter(s.User.ConnectedServer, aiID); enemy != nil {

		if enemy == nil || !enemy.IsActive {
			goto OUT
		}

		dmg, err := c.CalculateDamageToPlayer(enemy, false)
		if err != nil {
			goto OUT
		}

		time.AfterFunc(time.Second/2, func() { // attack done
			if c.CanAttack(enemy) {
				go DealDamageToPlayer(s, enemy, dmg, false)
			}
		})
	}

OUT:
	resp := INST_ATTACKED
	resp[4] = data[4]
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.IntToBytes(uint64(aiID), 2, true), 9)       // ai id

	p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: resp, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		return nil, err
	}
	if st.PoisonATK > 0 {
		//go ai.Poison(st.PoisonATK, c)
	}
	if st.ParalysisATK > 0 {
		//go ai.Paralysis(st.ParalysisATK)
	}
	if st.ConfusionATK > 0 {
		//go ai.Confusion(st.ConfusionATK)
	}

	c.AttackDelay = 1
	defer time.AfterFunc(time.Second*1, func() {
		c.AttackDelay = 0
	})
	return resp, nil
}

func (h *DealDamageHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character

	if c == nil {
		return nil, nil
	}

	resp := utils.Packet{}
	if c.TamingAI != nil {
		ai := c.TamingAI
		pos := database.GetNPCPosByID(ai.PosID)
		if pos == nil {
			return nil, nil
		}
		npc, ok := database.GetNpcInfo(pos.NPCID)
		if !ok {
			return nil, nil
		}
		petInfo := database.Pets[int64(npc.ID)]

		seed := utils.RandInt(0, 1000)
		//proportion := float64(ai.HP) / float64(npc.MaxHp) proportion < 0.1 &&
		if seed < 460 && petInfo != nil {
			go c.DealDamage(ai, ai.HP, true)

			item := &database.InventorySlot{ItemID: int64(npc.ID), Quantity: 1}
			expInfo := database.PetExps[petInfo.Level-1]
			item.Pet = &database.PetSlot{
				Fullness: 100, Loyalty: 100,
				Exp:   float64(expInfo.ReqExpEvo1),
				HP:    petInfo.BaseHP,
				Level: byte(petInfo.Level),
				Name:  petInfo.Name,
				CHI:   petInfo.BaseChi,
			}

			r, _, err := s.Character.AddItem(item, -1, true)
			if err != nil {
				return nil, err
			}

			resp.Concat(*r)
		}

		c.TamingAI = nil
		return resp, nil
	}

	targets := c.Targets
	dealt := make(map[int]struct{})
	for _, target := range targets {
		if target == nil {
			continue
		}

		ai := target.AI
		if _, ok := dealt[ai.ID]; ok {
			continue
		}

		dmg := target.Damage

		if target.SkillId != 0 {
			c.DealInfection(ai, nil, target.SkillId)
			go c.DealDamage(ai, dmg, true)
		} else {
			go c.DealDamage(ai, dmg, false)
		}
		dealt[ai.ID] = struct{}{}

		slots, err := c.InventorySlots()
		resp := utils.Packet{}
		if err == nil {
			weapon := slots[c.WeaponSlot]
			if weapon.ItemID != 0 {
				weaponinfo, _ := database.GetItemInfo(weapon.ItemID)
				if weaponinfo.Type == 105 {
					if slots[3].ItemID != 0 && slots[4].ItemID != 0 {
						slot3info, _ := database.GetItemInfo(slots[3].ItemID)
						slot4info, _ := database.GetItemInfo(slots[4].ItemID)
						if slot3info.Type == 106 {
							resp.Concat(*c.DecrementItem(3, 1))
							if slots[3].ItemID == 0 {
								gsh, err := c.GetStats()
								if err != nil {
									return nil, err
								}
								resp.Concat(gsh)
								slot, _, err := c.FindItemInInventory(nil, 10810001, 10810002, 10810003, 10810004)
								if err == nil && slot != -1 {
									swap, err := c.SwapItems(slot, 3)
									if err == nil {
										resp.Concat(swap)
									}
								}

							}
						} else if slot4info.Type == 106 {
							resp.Concat(*c.DecrementItem(4, 1))
							if slots[4].ItemID == 0 {
								gsh, err := c.GetStats()
								if err != nil {
									return nil, err
								}
								resp.Concat(gsh)
								slot, _, err := c.FindItemInInventory(nil, 10810001, 10810002, 10810003, 10810004)
								if err == nil && slot != -1 {
									swap, err := c.SwapItems(slot, 4)
									if err == nil {
										resp.Concat(swap)
									}
								}

							}
						}
						c.Socket.Write(resp)
					}
				}
			}

		}
		if ai.Map == 233 { //disable aoe in temples map
			break
		}
	}

	pTargets := c.PlayerTargets
	dealt = make(map[int]struct{})
	for _, target := range pTargets {
		if target == nil {
			continue
		}

		enemy := target.Enemy
		if _, ok := dealt[enemy.ID]; ok {
			continue
		}

		if c.CanAttack(enemy) {
			dmg := target.Damage
			if target.SkillId != 0 {
				go DealDamageToPlayer(s, enemy, dmg, true)
			} else {
				go DealDamageToPlayer(s, enemy, dmg, false)
			}

		}

		dealt[enemy.ID] = struct{}{}

		slots, err := c.InventorySlots()
		resp := utils.Packet{}
		if err == nil {
			weapon := slots[c.WeaponSlot]
			if weapon.ItemID != 0 {
				weaponinfo, _ := database.GetItemInfo(weapon.ItemID)
				if weaponinfo.Type == 105 {
					if slots[3].ItemID != 0 && slots[4].ItemID != 0 {
						slot3info, _ := database.GetItemInfo(slots[3].ItemID)
						slot4info, _ := database.GetItemInfo(slots[4].ItemID)
						if slot3info.Type == 106 {
							resp.Concat(*c.DecrementItem(3, 1))
							if slots[3].ItemID == 0 {
								gsh, err := c.GetStats()
								if err != nil {
									return nil, err
								}
								resp.Concat(gsh)
								slot, _, err := c.FindItemInInventory(nil, 10810001, 10810002, 10810003, 10810004)
								if err == nil && slot != -1 {
									swap, err := c.SwapItems(slot, 3)
									if err == nil {
										resp.Concat(swap)
									}
								}

							}
						} else if slot4info.Type == 106 {
							resp.Concat(*c.DecrementItem(4, 1))
							if slots[4].ItemID == 0 {
								gsh, err := c.GetStats()
								if err != nil {
									return nil, err
								}
								resp.Concat(gsh)
								slot, _, err := c.FindItemInInventory(nil, 10810001, 10810002, 10810003, 10810004)
								if err == nil && slot != -1 {
									swap, err := c.SwapItems(slot, 4)
									if err == nil {
										resp.Concat(swap)
									}
								}

							}
						}
						c.Socket.Write(resp)
					}
				}
			}

		}
	}

	c.Targets = []*database.Target{}
	c.PlayerTargets = []*database.PlayerTarget{}
	return nil, nil
}
func DealDamageToPlayer(s *database.Socket, enemy *database.Character, dmg int, isSkill bool) {
	r := PVP_DEAL_DAMAGE
	if isSkill {
		r = PVP_DEAL_SKILL_DAMAGE
	}

	c := s.Character
	enemySt := enemy.Socket.Stats
	st := s.Stats

	if c == nil {
		log.Println("character is nil")
		return
	} else if enemySt.HP <= 0 {
		return
	}
	stat := c.Socket.Stats
	seed := utils.RandInt(0, 1000) ///CRITICAL CHANCE
	if seed <= int64(stat.CriticalProbability) {
		critical := dmg + int(float32(dmg)*(float32(stat.CriticalRate)/100))
		dmg = critical
		if isSkill {
			r = PVP_DEAL_SKILL_CRITICAL_DAMAGE
		} else {
			r = PVP_DEAL_CRITICAL_DAMAGE
		}
	}

	if s.Character.Invisible {
		for _, invskillID := range database.InvisibilitySkillIDs {
			buff, _ := database.FindBuffByID(invskillID, c.ID)
			if buff != nil {
				buff.Duration = 0
				buff.Update()
			}
		}

	}

	if enemy.Meditating == true { //STOP MEDITATION
		enemy.Meditating = false
		med := MEDITATION_MODE
		med.Insert(utils.IntToBytes(uint64(enemy.PseudoID), 2, true), 6) // character pseudo id
		med[8] = 0

		p := nats.CastPacket{CastNear: true, CharacterID: enemy.ID, Type: nats.MEDITATION_MODE, Data: med}
		if err := p.Cast(); err == nil {
			enemy.Socket.Write(med)
		}
	}
	if enemy.HpRecoveryCooldown <= 0 {
		enemy.HpRecoveryCooldown += 2
	}

	enemySt.HP -= dmg
	if enemySt.HP < 0 {
		enemySt.HP = 0
	}

	r.Insert(utils.IntToBytes(uint64(enemy.PseudoID), 2, true), 5) // character pseudo id
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 7)     // mob pseudo id
	r.Insert(utils.IntToBytes(uint64(enemySt.HP), 4, true), 9)     // character hp
	r.Insert(utils.IntToBytes(uint64(enemySt.CHI), 4, true), 13)   // character chi

	r.Concat(enemy.GetHPandChi())
	p := &nats.CastPacket{CastNear: true, CharacterID: enemy.ID, Data: r, Type: nats.PLAYER_ATTACK}
	if err := p.Cast(); err != nil {
		log.Println("deal damage broadcast error:", err)
		return
	}

	if enemySt.HP <= 0 {
		enemySt.HP = 0
		enemy.KilledByCharacter = c
		enemy.Socket.Write(enemy.GetHPandChi())
		info := fmt.Sprintf("[%s] has defeated [%s]", c.Name, enemy.Name)
		r := messaging.InfoMessage(info)

		if enemy.Map != 255 && enemy.Map != 230 && enemySt.Honor >= 10 {
			c.Socket.Stats.Honor += 10
			c.Socket.Stats.Update()
			enemySt.Honor -= 10
			enemySt.Update()
			c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You acquired 10 Honor points.")))
			enemy.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have lost 10 Honor points.")))
			stat, _ := c.GetStats()
			c.Socket.Write(stat)
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: r, Type: nats.PVP_FINISHED}
		p.Cast()

		if enemy.Map == 255 && database.IsFactionWarStarted() {
			if enemy.Faction == 1 && c.Faction == 2 {
				database.AddPointsToFactionWarFaction(15, 2)
			}
			if enemy.Faction == 2 && c.Faction == 1 {
				database.AddPointsToFactionWarFaction(15, 1)
			}
			c.WarContribution += 15
			c.WarKillCount++
		} else if enemy.Map == 249 && database.IsFlagKingdomStarted() {
			if enemy.Faction == 1 && c.Faction == 2 {
				database.AddPointsToFlagKingdomFaction(5, 2)
			}
			if enemy.Faction == 2 && c.Faction == 1 {
				database.AddPointsToFlagKingdomFaction(5, 1)
			}
			c.WarContribution += 5
			c.WarKillCount++
		}

	}
	if st.ParalysisATK > 0 {
		//go enemy.Paralysis(st.ParalysisATK, enemy)
	}
	if st.ConfusionATK > 0 {
		//go enemy.Confusion(st.ConfusionATK, enemy)
	}
	if st.PoisonATK > 0 {
		//go enemy.Poison(st.PoisonATK, enemy)
	}
}

func (h *CastSkillHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	cX := 0.0
	cY := 0.0
	cZ := 0.0
	targetID := 0
	if len(data) >= 25 {
		cX = utils.BytesToFloat(data[11:15], true)
		cY = utils.BytesToFloat(data[15:19], true)
		cZ = utils.BytesToFloat(data[19:23], true)
		targetID = int(utils.BytesToInt(data[23:25], true))
	}

	attackCounter := int(data[6])
	skillID := int(utils.BytesToInt(data[7:11], true))

	return s.Character.CastSkill(attackCounter, skillID, targetID, cX, cY, cZ)
}

func (h *CastMonkSkillHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	cX := 0.0
	cY := 0.0
	cZ := 0.0
	targetID := 0
	if len(data) >= 24 {
		cX = utils.BytesToFloat(data[10:14], true)
		cY = utils.BytesToFloat(data[14:18], true)
		cZ = utils.BytesToFloat(data[18:22], true)
		targetID = int(utils.BytesToInt(data[22:24], true))
	}

	skillID := int(utils.BytesToInt(data[6:10], true))

	resp := utils.Packet{0xAA, 0x55, 0x16, 0x00, 0x49, 0x10, 0x55, 0xAA}
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // character pseudo id
	resp.Insert(utils.FloatToBytes(cX, 4, true), 8)                         // coordinate-x
	resp.Insert(utils.FloatToBytes(cY, 4, true), 12)                        // coordinate-y
	resp.Insert(utils.FloatToBytes(cZ, 4, true), 16)                        // coordinate-z
	resp.Insert(utils.IntToBytes(uint64(targetID), 2, true), 20)            // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(skillID), 4, true), 22)             // skill id

	return resp, nil
}

func (h *RemoveBuffHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	infectionID := int(utils.BytesToInt(data[6:10], true))
	buff, err := database.FindBuffByID(infectionID, s.Character.ID)
	if err != nil {
		return nil, err
	} else if buff == nil {
		return nil, nil
	}

	buff.Duration = 0
	buff.Update()
	return nil, nil
}
