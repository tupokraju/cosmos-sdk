package mempool_test

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

func TestOutOfOrder(t *testing.T) {
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 2)
	sa := accounts[0].Address
	sb := accounts[1].Address

	outOfOrders := [][]testTx{
		{
			{priority: 20, nonce: 1, address: sa},
			{priority: 21, nonce: 4, address: sa},
			{priority: 15, nonce: 1, address: sb},
			{priority: 8, nonce: 3, address: sa},
			{priority: 6, nonce: 2, address: sa},
		},
		{
			{priority: 15, nonce: 1, address: sb},
			{priority: 20, nonce: 1, address: sa},
			{priority: 21, nonce: 4, address: sa},
			{priority: 8, nonce: 3, address: sa},
			{priority: 6, nonce: 2, address: sa},
		}}

	for _, outOfOrder := range outOfOrders {
		var mtxs []sdk.Tx
		for _, mtx := range outOfOrder {
			mtxs = append(mtxs, mtx)
		}
		err := validateOrder(mtxs)
		require.Error(t, err)
	}

	seed := time.Now().UnixNano()
	t.Logf("running with seed: %d", seed)
	randomTxs := genRandomTxs(seed, 1000, 10)
	var rmtxs []sdk.Tx
	for _, rtx := range randomTxs {
		rmtxs = append(rmtxs, rtx)
	}

	require.Error(t, validateOrder(rmtxs))

}

func (s *MempoolTestSuite) TestPriorityNonceTxOrder() {
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 5)
	sa := accounts[0].Address
	sb := accounts[1].Address
	sc := accounts[2].Address

	tests := []struct {
		txs   []txSpec
		order []int
		fail  bool
	}{
		{
			txs: []txSpec{
				{p: 21, n: 4, a: sa},
				{p: 8, n: 3, a: sa},
				{p: 6, n: 2, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 20, n: 1, a: sa},
			},
			order: []int{4, 3, 2, 1, 0},
		},
		{
			txs: []txSpec{
				{p: 3, n: 0, a: sa},
				{p: 5, n: 1, a: sa},
				{p: 9, n: 2, a: sa},
				{p: 6, n: 0, a: sb},
				{p: 5, n: 1, a: sb},
				{p: 8, n: 2, a: sb},
			},
			order: []int{3, 4, 5, 0, 1, 2},
		},
		{
			txs: []txSpec{
				{p: 21, n: 4, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 20, n: 1, a: sa},
			},
			order: []int{2, 0, 1},
		},
		{
			txs: []txSpec{
				{p: 50, n: 3, a: sa},
				{p: 30, n: 2, a: sa},
				{p: 10, n: 1, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 21, n: 2, a: sb},
			},
			order: []int{3, 4, 2, 1, 0},
		},
		{
			txs: []txSpec{
				{p: 50, n: 3, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 99, n: 1, a: sa},
				{p: 15, n: 1, a: sb},
				{p: 8, n: 2, a: sb},
			},
			order: []int{2, 3, 1, 0, 4},
		},
		{
			txs: []txSpec{
				{p: 30, a: sa, n: 2},
				{p: 20, a: sb, n: 1},
				{p: 15, a: sa, n: 1},
				{p: 10, a: sa, n: 0},
				{p: 8, a: sb, n: 0},
				{p: 6, a: sa, n: 3},
				{p: 4, a: sb, n: 3},
			},
			order: []int{3, 2, 0, 4, 1, 5, 6},
		},
		{
			txs: []txSpec{
				{p: 30, n: 2, a: sa},
				{p: 20, a: sb, n: 1},
				{p: 15, a: sa, n: 1},
				{p: 10, a: sa, n: 0},
				{p: 8, a: sb, n: 0},
				{p: 6, a: sa, n: 3},
				{p: 4, a: sb, n: 3},
				{p: 2, a: sc, n: 0},
				{p: 7, a: sc, n: 3},
			},
			order: []int{3, 2, 0, 4, 1, 5, 6, 7, 8},
		},
		{
			txs: []txSpec{
				{p: 6, n: 1, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 5, n: 1, a: sb},
				{p: 99, n: 2, a: sb},
			},
			order: []int{0, 1, 2, 3},
		},
		{
			// if all txs have the same priority they will be ordered lexically sender address, and nonce with the
			// sender.
			txs: []txSpec{
				{p: 10, n: 7, a: sc},
				{p: 10, n: 8, a: sc},
				{p: 10, n: 9, a: sc},
				{p: 10, n: 1, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 10, n: 3, a: sa},
				{p: 10, n: 4, a: sb},
				{p: 10, n: 5, a: sb},
				{p: 10, n: 6, a: sb},
			},
			order: []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
		},
		/*
			The next 4 tests are different permutations of the same set:

			  		{p: 5, n: 1, a: sa},
					{p: 10, n: 2, a: sa},
					{p: 20, n: 2, a: sb},
					{p: 5, n: 1, a: sb},
					{p: 99, n: 2, a: sc},
					{p: 5, n: 1, a: sc},

			which exercises the actions required to resolve priority ties.
		*/
		{
			txs: []txSpec{
				{p: 5, n: 1, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 5, n: 1, a: sb},
				{p: 99, n: 2, a: sb},
			},
			order: []int{2, 3, 0, 1},
		},
		{
			txs: []txSpec{
				{p: 5, n: 1, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 20, n: 2, a: sb},
				{p: 5, n: 1, a: sb},
				{p: 99, n: 2, a: sc},
				{p: 5, n: 1, a: sc},
			},
			order: []int{5, 4, 3, 2, 0, 1},
		},
		{
			txs: []txSpec{
				{p: 5, n: 1, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 5, n: 1, a: sb},
				{p: 20, n: 2, a: sb},
				{p: 5, n: 1, a: sc},
				{p: 99, n: 2, a: sc},
			},
			order: []int{4, 5, 2, 3, 0, 1},
		},
		{
			txs: []txSpec{
				{p: 5, n: 1, a: sa},
				{p: 10, n: 2, a: sa},
				{p: 5, n: 1, a: sc},
				{p: 20, n: 2, a: sc},
				{p: 5, n: 1, a: sb},
				{p: 99, n: 2, a: sb},
			},
			order: []int{4, 5, 2, 3, 0, 1},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			pool := mempool.NewPriorityMempool()

			// create test txs and insert into mempool
			for i, ts := range tt.txs {
				tx := testTx{id: i, priority: int64(ts.p), nonce: uint64(ts.n), address: ts.a}
				c := ctx.WithPriority(tx.priority)
				err := pool.Insert(c, tx)
				require.NoError(t, err)
			}

			orderedTxs := fetchTxs(pool.Select(ctx, nil), 1000)
			var txOrder []int
			for _, tx := range orderedTxs {
				txOrder = append(txOrder, tx.(testTx).id)
				fmt.Println(tx)
			}
			require.Equal(t, tt.order, txOrder)
			require.NoError(t, validateOrder(orderedTxs))

			for _, tx := range orderedTxs {
				require.NoError(t, pool.Remove(tx))
			}

			require.NoError(t, mempool.IsEmpty(pool))
		})
	}
}

func (s *MempoolTestSuite) TestPriorityTies() {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 3)
	sa := accounts[0].Address
	sb := accounts[1].Address
	sc := accounts[2].Address

	txSet := []txSpec{
		{p: 5, n: 1, a: sc},
		{p: 99, n: 2, a: sc},
		{p: 5, n: 1, a: sb},
		{p: 20, n: 2, a: sb},
		{p: 5, n: 1, a: sa},
		{p: 10, n: 2, a: sa},
	}

	for i := 0; i < 100; i++ {
		s.mempool = mempool.NewPriorityMempool()
		var shuffled []txSpec
		for _, t := range txSet {
			tx := txSpec{
				p: t.p,
				n: t.n,
				a: t.a,
			}
			shuffled = append(shuffled, tx)
		}
		rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

		for id, ts := range shuffled {
			tx := testTx{priority: int64(ts.p), nonce: uint64(ts.n), address: ts.a, id: id}
			c := ctx.WithPriority(tx.priority)
			err := s.mempool.Insert(c, tx)
			s.NoError(err)
		}
		selected := fetchTxs(s.mempool.Select(ctx, nil), 1000)
		var orderedTxs []txSpec
		for _, tx := range selected {
			ttx := tx.(testTx)
			ts := txSpec{p: int(ttx.priority), n: int(ttx.nonce), a: ttx.address}
			orderedTxs = append(orderedTxs, ts)
		}
		s.Equal(txSet, orderedTxs)
	}
}

func (s *MempoolTestSuite) TestRandomTxOrderManyTimes() {
	for i := 0; i < 3; i++ {
		s.Run("TestRandomGeneratedTxs", func() {
			s.TestRandomGeneratedTxs()
		})
		s.Run("TestRandomWalkTxs", func() {
			s.TestRandomWalkTxs()
		})
	}
}

// validateOrder checks that the txs are ordered by priority and nonce
// in O(n^2) time by checking each tx against all the other txs
func validateOrder(mtxs []sdk.Tx) error {
	iterations := 0
	var itxs []txSpec
	for i, mtx := range mtxs {
		iterations++
		tx := mtx.(testTx)
		itxs = append(itxs, txSpec{p: int(tx.priority), n: int(tx.nonce), a: tx.address, i: i})
	}

	// Given 2 transactions t1 and t2, where t2.p > t1.p but t2.i < t1.i
	// Then if t2.sender have the same sender then t2.nonce > t1.nonce
	// or
	// If t1 and t2 have different senders then there must be some t3 with
	// t3.sender == t2.sender and t3.n < t2.n and t3.p <= t1.p

	for _, a := range itxs {
		for _, b := range itxs {
			iterations++
			// when b is before a

			// when a is before b
			if a.i < b.i {
				// same sender
				if a.a.Equals(b.a) {
					// same sender
					if a.n == b.n {
						return fmt.Errorf("same sender tx have the same nonce\n%v\n%v", a, b)
					}
					if a.n > b.n {
						return fmt.Errorf("same sender tx have wrong nonce order\n%v\n%v", a, b)
					}
				} else {
					// different sender
					if a.p < b.p {
						// find a tx with same sender as b and lower nonce
						found := false
						for _, c := range itxs {
							iterations++
							if c.a.Equals(b.a) && c.n < b.n && c.p <= a.p {
								found = true
								break
							}
						}
						if !found {
							return fmt.Errorf("different sender tx have wrong order\n%v\n%v", b, a)
						}
					}
				}
			}
		}
	}
	// fmt.Printf("validation in iterations: %d\n", iterations)
	return nil
}

func (s *MempoolTestSuite) TestRandomGeneratedTxs() {
	s.iterations = 0
	s.mempool = mempool.NewPriorityMempool(mempool.WithOnRead(func(tx sdk.Tx) {
		s.iterations++
	}))
	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	seed := time.Now().UnixNano()

	t.Logf("running with seed: %d", seed)
	generated := genRandomTxs(seed, s.numTxs, s.numAccounts)
	mp := s.mempool

	for _, otx := range generated {
		tx := testTx{id: otx.id, priority: otx.priority, nonce: otx.nonce, address: otx.address}
		c := ctx.WithPriority(tx.priority)
		err := mp.Insert(c, tx)
		require.NoError(t, err)
	}

	selected := fetchTxs(mp.Select(ctx, nil), 100000)
	for i, tx := range selected {
		ttx := tx.(testTx)
		sigs, _ := tx.(signing.SigVerifiableTx).GetSignaturesV2()
		ttx.strAddress = sigs[0].PubKey.Address().String()
		selected[i] = ttx
	}
	require.Equal(t, len(generated), len(selected))

	start := time.Now()
	require.NoError(t, validateOrder(selected))
	duration := time.Since(start)

	fmt.Printf("seed: %d completed in %d iterations; validation in %dms\n",
		seed, s.iterations, duration.Milliseconds())
}

func (s *MempoolTestSuite) TestRandomWalkTxs() {
	s.iterations = 0
	s.mempool = mempool.NewPriorityMempool()

	t := s.T()
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())

	seed := time.Now().UnixNano()
	// interesting failing seeds:
	// seed := int64(1663971399133628000)
	// seed := int64(1663989445512438000)
	//
	t.Logf("running with seed: %d", seed)

	ordered, shuffled := genOrderedTxs(seed, s.numTxs, s.numAccounts)
	mp := s.mempool

	for _, otx := range shuffled {
		tx := testTx{id: otx.id, priority: otx.priority, nonce: otx.nonce, address: otx.address}
		c := ctx.WithPriority(tx.priority)
		err := mp.Insert(c, tx)
		require.NoError(t, err)
	}

	require.Equal(t, s.numTxs, mp.CountTx())

	selected := fetchTxs(mp.Select(ctx, nil), math.MaxInt)
	require.Equal(t, len(ordered), len(selected))
	var orderedStr, selectedStr string

	for i := 0; i < s.numTxs; i++ {
		otx := ordered[i]
		stx := selected[i].(testTx)
		orderedStr = fmt.Sprintf("%s\n%s, %d, %d; %d",
			orderedStr, otx.address, otx.priority, otx.nonce, otx.id)
		selectedStr = fmt.Sprintf("%s\n%s, %d, %d; %d",
			selectedStr, stx.address, stx.priority, stx.nonce, stx.id)
	}

	require.Equal(t, s.numTxs, len(selected))

	errMsg := fmt.Sprintf("Expected order: %v\nGot order: %v\nSeed: %v", orderedStr, selectedStr, seed)

	start := time.Now()
	require.NoError(t, validateOrder(selected), errMsg)
	duration := time.Since(start)

	t.Logf("seed: %d completed in %d iterations; validation in %dms\n",
		seed, s.iterations, duration.Milliseconds())
}

func genRandomTxs(seed int64, countTx int, countAccount int) (res []testTx) {
	maxPriority := 100
	r := rand.New(rand.NewSource(seed))
	accounts := simtypes.RandomAccounts(r, countAccount)
	accountNonces := make(map[string]uint64)
	for _, account := range accounts {
		accountNonces[account.Address.String()] = 0
	}

	for i := 0; i < countTx; i++ {
		addr := accounts[r.Intn(countAccount)].Address
		priority := int64(r.Intn(maxPriority + 1))
		nonce := accountNonces[addr.String()]
		accountNonces[addr.String()] = nonce + 1
		res = append(res, testTx{
			priority: priority,
			nonce:    nonce,
			address:  addr,
			id:       i})
	}

	return res
}

// since there are multiple valid ordered graph traversals for a given set of txs strict
// validation against the ordered txs generated from this function is not possible as written
func genOrderedTxs(seed int64, maxTx int, numAcc int) (ordered []testTx, shuffled []testTx) {
	r := rand.New(rand.NewSource(seed))
	accountNonces := make(map[string]uint64)
	prange := 10
	randomAccounts := simtypes.RandomAccounts(r, numAcc)
	for _, account := range randomAccounts {
		accountNonces[account.Address.String()] = 0
	}

	getRandAccount := func(notAddress string) simtypes.Account {
		for {
			res := randomAccounts[r.Intn(len(randomAccounts))]
			if res.Address.String() != notAddress {
				return res
			}
		}
	}

	txCursor := int64(10000)
	ptx := testTx{address: getRandAccount("").Address, nonce: 0, priority: txCursor}
	samepChain := make(map[string]bool)
	for i := 0; i < maxTx; {
		var tx testTx
		move := r.Intn(5)
		switch move {
		case 0:
			// same sender, less p
			nonce := ptx.nonce + 1
			tx = testTx{nonce: nonce, address: ptx.address, priority: txCursor - int64(r.Intn(prange)+1)}
			txCursor = tx.priority
		case 1:
			// same sender, same p
			nonce := ptx.nonce + 1
			tx = testTx{nonce: nonce, address: ptx.address, priority: ptx.priority}
		case 2:
			// same sender, greater p
			nonce := ptx.nonce + 1
			tx = testTx{nonce: nonce, address: ptx.address, priority: ptx.priority + int64(r.Intn(prange)+1)}
		case 3:
			// different sender, less p
			sender := getRandAccount(ptx.address.String()).Address
			nonce := accountNonces[sender.String()] + 1
			tx = testTx{nonce: nonce, address: sender, priority: txCursor - int64(r.Intn(prange)+1)}
			txCursor = tx.priority
		case 4:
			// different sender, same p
			sender := getRandAccount(ptx.address.String()).Address
			// disallow generating cycles of same p txs. this is an invalid processing order according to our
			// algorithm decision.
			if _, ok := samepChain[sender.String()]; ok {
				continue
			}
			nonce := accountNonces[sender.String()] + 1
			tx = testTx{nonce: nonce, address: sender, priority: txCursor}
			samepChain[sender.String()] = true
		}
		tx.id = i
		accountNonces[tx.address.String()] = tx.nonce
		ordered = append(ordered, tx)
		ptx = tx
		i++
		if move != 4 {
			samepChain = make(map[string]bool)
		}
	}

	for _, item := range ordered {
		tx := testTx{
			priority: item.priority,
			nonce:    item.nonce,
			address:  item.address,
			id:       item.id,
		}
		shuffled = append(shuffled, tx)
	}
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return ordered, shuffled
}

func TestTxOrderN(t *testing.T) {
	numTx := 10

	seed := time.Now().UnixNano()
	ordered, shuffled := genOrderedTxs(seed, numTx, 3)
	require.Equal(t, numTx, len(ordered))
	require.Equal(t, numTx, len(shuffled))

	fmt.Println("ordered")
	for _, tx := range ordered {
		fmt.Printf("%s, %d, %d\n", tx.address, tx.priority, tx.nonce)
	}

	fmt.Println("shuffled")
	for _, tx := range shuffled {
		fmt.Printf("%s, %d, %d\n", tx.address, tx.priority, tx.nonce)
	}
}
