package main

import (
	"fmt"
	"time"

	"github.com/bucloud/hwapi"
)

func (as *accountSpace) getAccounts() []*hwapi.SimpleAccount {
	a := as.Accounts.Load()
	if a != nil {
		return a.([]*hwapi.SimpleAccount)
	}
	return []*hwapi.SimpleAccount{}
}
func (as *accountSpace) setAccounts(a []*hwapi.SimpleAccount) {
	as.Accounts.Store(a)
}

func simpleAccounts(accounts *hwapi.Account) []*hwapi.SimpleAccount {
	res := []*hwapi.SimpleAccount{}
	res = append(res, &hwapi.SimpleAccount{
		ID:            fmt.Sprintf("%d", accounts.Id),
		AccountName:   accounts.AccountName,
		AccountHash:   accounts.AccountHash,
		AccountStatus: accounts.AccountStatus,
	})
	for _, sa := range accounts.SubAccounts {
		res = append(res, simpleAccounts(sa)...)
	}
	return res
}

// getAccountStatusByAccountHash return account status
func (as *accountSpace) getAccountByAccountHash(ah string) *hwapi.SimpleAccount {
	for _, a := range as.getAccounts() {
		if a.AccountHash == ah {
			return a
		}
	}
	return nil
}

func (as *accountSpace) accountsSync() {
	wg.Add(1)
	for {
		tempStartTimeStamp := time.Now().Unix()
		subAccounts, e := as.API.GetSubaccounts(as.SuperAccount, "true")
		if e != nil {
			as.log.Println(e.Error())
		} else {
			accounts := simpleAccounts(subAccounts)
			as.setAccounts(accounts)
			as.log.Printf("Get %d Accounts list succeed", len(accounts))
		}

		currentTimeStamp := time.Now().Unix()
		as.log.Printf("Last account sync spent %d second, next round will start after %d second", currentTimeStamp-tempStartTimeStamp, 20*60-currentTimeStamp+tempStartTimeStamp)
		time.Sleep(time.Duration(20*60-currentTimeStamp+tempStartTimeStamp) * time.Second)
	}
}
