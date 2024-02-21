package highs_test

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nextmv-io/go-highs"
	"github.com/nextmv-io/go-mip"
	mipModel "github.com/nextmv-io/go-mip/model"
)

func TestHighs(t *testing.T) {
	testSolver(func(m mip.Model) (mip.Solver, error) {
		return highs.NewSolver(m), nil
	}, t)
}

func TestHighsMIQP(t *testing.T) {
	// minimize x_1^2
	//
	// subject to x integer
	// should error for HiGHS
	m := mip.NewModel()
	x1 := m.NewInt(4, math.MaxInt64)

	obj := m.Objective()
	obj.SetMinimize()
	obj.NewQuadraticTerm(1.0, x1, x1)
	solver := highs.NewSolver(m)
	opt := mip.SolveOptions{}
	_, err := solver.Solve(opt)
	if err == nil {
		t.Error("Want error, got nil")
	}
}

func TestHighsMIQP2(t *testing.T) {
	// minimize x_2^2
	//
	// subject to x1 <= x2, x2 integer
	// should error for HiGHS
	m := mip.NewModel()
	x1 := m.NewFloat(4, math.MaxFloat64)
	x2 := m.NewInt(4, math.MaxInt64)

	obj := m.Objective()
	obj.SetMinimize()
	obj.NewQuadraticTerm(1.0, x2, x2)
	cstr := m.NewConstraint(mip.LessThanOrEqual, 0)
	cstr.NewTerm(1, x1)
	cstr.NewTerm(-1, x2)
	solver := highs.NewSolver(m)
	opt := mip.SolveOptions{}
	_, err := solver.Solve(opt)
	if err == nil {
		t.Error("Want error, got nil")
	}
}

type solverFactory func(mip.Model) (mip.Solver, error)

func defaultOptions() mip.SolveOptions {
	return mip.SolveOptions{
		Duration:  10 * time.Second,
		Verbosity: mip.Off,
		MIP: mip.MIPOptions{
			Gap: mip.GapOptions{
				Absolute: 0.0,
				Relative: 0.0,
			},
		},
	}
}

func testSolver(s solverFactory, t *testing.T) {
	t.Run("EmptyModelTest", func(t *testing.T) {
		emptyModelTest(s, t)
	})
	t.Run("NoObjectiveTest", func(t *testing.T) {
		noObjectiveTest(s, t)
	})
	t.Run("NoConstraintTermsTest", func(t *testing.T) {
		noConstraintTermsTest(s, t)
	})
	t.Run("ZeroCoefficientConstraintTermTest", func(t *testing.T) {
		zeroCoefficientConstraintTermTest(s, t)
	})
	t.Run("ZeroCoefficientObjectiveTermTest", func(t *testing.T) {
		zeroCoefficientObjectiveTermTest(s, t)
	})
	t.Run("SingleBoolMinimizeTest", func(t *testing.T) {
		singleVarTest(boolVar, true, s, t)
	})
	t.Run("SingleBoolMaximizeTest", func(t *testing.T) {
		singleVarTest(boolVar, false, s, t)
	})
	t.Run("SingleFloatMinimizeTest", func(t *testing.T) {
		singleVarTest(floatVar, true, s, t)
	})
	t.Run("SingleFloatMaximizeTest", func(t *testing.T) {
		singleVarTest(floatVar, false, s, t)
	})
	t.Run("SingleIntMinimizeTest", func(t *testing.T) {
		singleVarTest(intVar, true, s, t)
	})
	t.Run("SingleIntMaximizeTest", func(t *testing.T) {
		singleVarTest(intVar, false, s, t)
	})
	t.Run("Binkies", func(t *testing.T) {
		binkiesTest(s, t)
	})
	t.Run("BoundsTest", func(t *testing.T) {
		boundsTest(s, t)
	})
	t.Run("SudokuTest", func(t *testing.T) {
		sudokuTest(s, t)
	})
	t.Run("QPTest1", func(t *testing.T) {
		qpTest1(s, t)
	})
	t.Run("QPTest2", func(t *testing.T) {
		qpTest2(s, t)
	})
	t.Run("QPTest3", func(t *testing.T) {
		qpTest3(s, t)
	})
	t.Run("linearRegression", func(t *testing.T) {
		linearRegression(s, t)
	})
	t.Run("QPPortfolio", func(t *testing.T) {
		qpPortfolioOptim(s, t)
	})
	t.Run("QpAnotherExample", func(t *testing.T) {
		qpAnotherExample(s, t)
	})
	t.Run("FloatLowerBoundNaNTest", func(t *testing.T) {
		floatBoundNaNTest(math.NaN(), 0.0, t)
	})
	t.Run("FloatUpperBoundNaNTest", func(t *testing.T) {
		floatBoundNaNTest(0.0, math.NaN(), t)
	})
	t.Run("ObjectiveTermCoefficientNaNTest", func(t *testing.T) {
		objectiveTermCoefficientNaNTest(t)
	})
	t.Run("ObjectiveQuadraticTermCoefficientNaNTest", func(t *testing.T) {
		objectiveQuadraticTermCoefficientNaNTest(t)
	})
	t.Run("ConstraintTermCoefficientNaNTest", func(t *testing.T) {
		constraintTermCoefficientNaNTest(t)
	})
}

func floatBoundNaNTest(lb, ub float64, t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic on NaN bound")
		}
	}()

	model := mip.NewModel()

	model.NewFloat(lb, ub)
}

func constraintTermCoefficientNaNTest(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic on NaN coefficient")
		}
	}()

	model := mip.NewModel()

	x := model.NewFloat(0, 100.0)

	cnstr := model.NewConstraint(mip.LessThanOrEqual, 100.0)
	cnstr.NewTerm(math.NaN(), x)
}

func objectiveTermCoefficientNaNTest(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic on NaN coefficient")
		}
	}()

	model := mip.NewModel()

	x := model.NewFloat(0, 100.0)

	model.Objective().NewTerm(math.NaN(), x)
}

func objectiveQuadraticTermCoefficientNaNTest(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic on NaN coefficient")
		}
	}()

	model := mip.NewModel()

	x := model.NewFloat(0, 100.0)

	model.Objective().NewQuadraticTerm(math.NaN(), x, x)
}

func emptyModelTest(s solverFactory, t *testing.T) {
	model := mip.NewModel()

	solver, err := s(model)
	if err != nil {
		t.Error(err)
	}

	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Error(err)
	}

	if !solution.IsOptimal() {
		t.Errorf("expected optimal solution for empty model")
	}
}

func noConstraintTermsTest(s solverFactory, t *testing.T) {
	definition := mip.NewModel()

	x := definition.NewBool()

	costConstraint := definition.NewConstraint(mip.LessThanOrEqual, 1)

	_ = costConstraint
	_ = x

	solver, _ := s(definition)

	options := defaultOptions()
	_, err := solver.Solve(options)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func zeroCoefficientConstraintTermTest(
	s solverFactory,
	t *testing.T,
) {
	model := mip.NewModel()
	x := model.NewBool()
	costConstraint := model.NewConstraint(mip.LessThanOrEqual, 1)
	costConstraint.NewTerm(0, x)

	solver, _ := s(model)

	options := defaultOptions()
	_, err := solver.Solve(options)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func zeroCoefficientObjectiveTermTest(
	s solverFactory,
	t *testing.T,
) {
	model := mip.NewModel()
	x := model.NewBool()
	model.Objective().NewTerm(0, x)

	solver, _ := s(model)

	options := defaultOptions()
	_, err := solver.Solve(options)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func noObjectiveTest(s solverFactory, t *testing.T) {
	model := mip.NewModel()

	model.NewFloat(0.0, 1.0)

	solver, _ := s(model)

	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !solution.IsOptimal() {
		t.Errorf("expected optimal solution for empty model")
	}

	if !solution.HasValues() {
		t.Errorf("expected hasValues to be true")
	}
}

type variableType int

const (
	intVar variableType = iota
	boolVar
	floatVar
)

func singleVarTest(
	vType variableType,
	minimize bool,
	s solverFactory,
	t *testing.T,
) {
	model := mip.NewModel()

	var v mip.Var
	var err error

	minimum := 0.1
	maximum := 1.1
	switch vType {
	case floatVar:
		v = model.NewFloat(minimum, maximum)
	case boolVar:
		minimum = 0
		maximum = 1
		v = model.NewBool()
	case intVar:
		minimum = -1
		maximum = 1
		v = model.NewInt(int64(minimum), int64(maximum))
	}

	if minimize {
		model.Objective().SetMinimize()
	} else {
		model.Objective().SetMaximize()
	}

	model.Objective().NewTerm(1.0, v)

	solver, err := s(model)
	if err != nil {
		t.Error(err)
	}

	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Error(err)
	}

	if !solution.IsOptimal() {
		t.Errorf("expected optimal solution for empty model")
	}

	if !solution.HasValues() {
		t.Errorf("expected hasValues to be true")
	}

	optimalValue := maximum

	if minimize {
		optimalValue = minimum
	}

	if solution.Value(v) != optimalValue {
		t.Errorf("expected optimal value 0.0 got %v", solution.Value(v))
	}
}

func boundsTest(s solverFactory, t *testing.T) {
	model := mip.NewModel()

	x := model.NewFloat(0, 10)
	y := model.NewInt(0, 5)
	c := model.NewConstraint(mip.LessThanOrEqual, 8)
	c.NewTerm(1, x)
	c.NewTerm(1, y)

	model.Objective().SetMaximize()

	model.Objective().NewTerm(1, x)
	model.Objective().NewTerm(2, y)

	solver, err := s(model)
	if err != nil {
		t.Error(err)
	}

	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Error(err)
	}

	if !solution.HasValues() {
		t.Errorf("expected to have a solution," +
			" solution hasValues returns false")
	}

	if int(solution.ObjectiveValue()) != 13 {
		t.Errorf("expected to have an objective of 13 got %v",
			int(solution.ObjectiveValue()))
	}
}

// Cell is a helper structure for the sudoku test.
type Cell struct {
	id, value int
}

// ID returns the id of the cell.
func (b Cell) ID() string {
	return strconv.Itoa(b.id)
}

// BinaryHelper is an int with an ID method to implement the Identifier[int]
// interface.
type BinaryHelper string

// ID returns the id of a BinaryHelper.
func (b BinaryHelper) ID() string {
	return string(b)
}

// Binarized is a helper type returned by the Binarize function. It can be
// queried for the Identifier belonging to a specific number.
type Binarized []mipModel.Identifier

// GetIdentifier will return the Identifier that belongs to a number.
func (b Binarized) GetIdentifier(number int) mipModel.Identifier {
	return b[number-1]
}

// Binarize will take a number and return a Binarized (slice of Identifiers).
func Binarize(number int) Binarized {
	returnList := make([]mipModel.Identifier, number)
	for i := 0; i < number; i++ {
		returnList[i] = BinaryHelper(strconv.Itoa(i))
	}
	return returnList
}

func sudokuTest(s solverFactory, t *testing.T) {
	input := [9][9]int{
		{0, 0, 0, 0, 0, 6, 0, 3, 0},
		{1, 5, 0, 0, 0, 0, 0, 0, 4},
		{0, 0, 6, 0, 0, 0, 0, 0, 0},

		{0, 0, 3, 0, 0, 8, 4, 0, 0},
		{2, 0, 0, 0, 0, 0, 0, 6, 0},
		{0, 0, 0, 0, 4, 9, 0, 2, 5},

		{0, 0, 0, 7, 0, 5, 0, 0, 0},
		{0, 7, 0, 0, 9, 0, 1, 0, 3},
		{0, 6, 0, 0, 0, 0, 8, 0, 0},
	}

	// define domain objects
	cells := make([][]Cell, len(input))
	// only needed if we wanted checks on the index of the MultiMap
	cellsList := make([]mipModel.Identifier, len(input))
	sections := make([][]Cell, len(input))
	columns := make([][]Cell, len(input))
	// modeling helper for binary formulation
	numbers := Binarize(len(input))

	counter := 0
	// initialize domain objects
	for r := 0; r < len(input); r++ {
		for c := 0; c < len(input); c++ {
			counter++
			cell := Cell{id: counter, value: input[r][c]}
			cellsList = append(cellsList, cell)
			cells[r] = append(cells[r], cell)
			columns[c] = append(columns[c], cell)
			sectionIndex := (r/3)*3 + c/3
			sections[sectionIndex] = append(sections[sectionIndex], cell)
		}
	}

	// initialize a new model
	m := mip.NewModel()
	x := mipModel.NewMultiMap(
		func(...mipModel.Identifier) mip.Bool {
			return m.NewBool()
		}, cellsList, numbers)

	for r, row := range input {
		for c := 0; c < len(input); c++ {
			cell := cells[r][c]

			// fixed values
			if cell.value != 0 { // the value is fixed
				constraint := m.NewConstraint(mip.Equal, 1)
				constraint.NewTerm(1, x.Get(cell, numbers.GetIdentifier(cell.value)))
			}

			// each cell can only have one value
			constraint := m.NewConstraint(mip.Equal, 1)
			for _, number := range numbers {
				constraint.NewTerm(1, x.Get(cell, number))
			}
		}

		// each number in a row needs to be unique
		for _, number := range numbers {
			constraint := m.NewConstraint(mip.Equal, 1)
			for c := range row {
				constraint.NewTerm(1, x.Get(cells[r][c], number))
			}
		}
	}

	// each number in a column needs to be unique
	for _, column := range columns {
		for _, number := range numbers {
			constraint := m.NewConstraint(mip.Equal, 1)
			for _, cell := range column {
				constraint.NewTerm(1, x.Get(cell, number))
			}
		}
	}

	// each number in a section needs to be unique
	for _, section := range sections {
		for _, number := range numbers {
			constraint := m.NewConstraint(mip.Equal, 1)
			for _, cell := range section {
				constraint.NewTerm(1, x.Get(cell, number))
			}
		}
	}

	solver, err := s(m)
	if err != nil {
		panic(err)
	}

	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		panic(err)
	}

	sb := strings.Builder{}
	if solution.HasValues() {
		for r, row := range input {
			rowValues := []string{}
			for c := range row {
				cell := cells[r][c]
				for _, number := range numbers {
					if solution.Value(x.Get(cell, number)) != 0 {
						value, _ := strconv.Atoi(number.ID())
						cell.value = value + 1
					}
				}
				rowValues = append(rowValues, fmt.Sprint(cell.value))
			}
			sb.WriteString(strings.Join(rowValues, " "))
		}
	} else {
		sb.WriteString("No solution")
	}

	expected := "7 4 8 9 1 6 5 3 2" +
		"1 5 9 3 2 7 6 8 4" +
		"3 2 6 8 5 4 7 1 9" +
		"5 9 3 2 6 8 4 7 1" +
		"2 1 4 5 7 3 9 6 8" +
		"6 8 7 1 4 9 3 2 5" +
		"4 3 1 7 8 5 2 9 6" +
		"8 7 5 6 9 2 1 4 3" +
		"9 6 2 4 3 1 8 5 7"

	actual := sb.String()
	if actual != expected {
		t.Errorf("expected to a solution of %v got %v", expected, actual)
	}
}

// Maximize binkies is a demo program in which we maximize the number of
// binkies our bunnies will execute by selecting their diet.
//
// A binky is when a bunny jumps straight up and quickly twists its hind end,
// head, or both. A bunny may binky because it is feeling happy or safe in its
// environment.
func binkiesTest(s solverFactory, t *testing.T) {
	/*
	   Diet A requirements are 1 carrot and 3 units of endive and generates a
	   happiness of 6 binkies.

	   Diet B requirements are 1 carrot and 2 units of endive and generates a
	   happiness of 5 binkies.

	   There are 5 carrots in storage and 12 units of endive.

	   The bunny wishes to maximize its binkies.

	   How many units of diet A and diet B should the bunny consume?
	   ```
	   max numberOfDietA * binkiesPerDietA + numberOfDietB * binkiesPerDietB

	   numberOfDietA * unitsOfEndivePerDietA +
	   numberOfDietB * unitsOfEndivePerDietB <=
	   unitsOfEndiveInStock

	   numberOfDietA * unitsOfCarrotPerDietA +
	   numberOfDietB * unitsOfCarrotPerDietB <=
	   unitsOfCarrotsInStock
	   ```
	*/
	binkiesPerDietA := 6.0
	binkiesPerDietB := 5.0

	unitsOfEndivePerDietA := 3.0
	unitsOfEndivePerDietB := 2.0

	unitsOfCarrotPerDietA := 1.0
	unitsOfCarrotPerDietB := 1.0

	unitsOfEndiveInStock := 12.0
	unitsOfCarrotsInStock := 5.0

	model := mip.NewModel()

	numberOfDietA := model.NewInt(
		0,
		100,
	)
	numberOfDietB := model.NewInt(
		0,
		100,
	)

	model.Objective().SetMaximize()

	model.Objective().NewTerm(
		binkiesPerDietA,
		numberOfDietA,
	)
	model.Objective().NewTerm(
		binkiesPerDietB,
		numberOfDietB,
	)

	endivesConstraint := model.NewConstraint(
		mip.LessThanOrEqual,
		unitsOfEndiveInStock,
	)

	endivesConstraint.NewTerm(
		unitsOfEndivePerDietA,
		numberOfDietA,
	)
	endivesConstraint.NewTerm(
		unitsOfEndivePerDietB,
		numberOfDietB,
	)

	carrotsConstraint := model.NewConstraint(
		mip.LessThanOrEqual,
		unitsOfCarrotsInStock,
	)

	carrotsConstraint.NewTerm(
		unitsOfCarrotPerDietA,
		numberOfDietA,
	)
	carrotsConstraint.NewTerm(
		unitsOfCarrotPerDietB,
		numberOfDietB,
	)

	solver, err := s(model)
	if err != nil {
		t.Error(err)
	}
	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Error(err)
	}

	if !solution.HasValues() {
		t.Errorf("expected to have a solution," +
			" solution hasValues returns false")
	}

	if int(solution.ObjectiveValue()) != 27 {
		t.Errorf("expected to have an objective of 27 got %v",
			int(solution.ObjectiveValue()))
	}
}

func qpTest1(s solverFactory, t *testing.T) {
	// From the HiGHS tests
	// minimize -x_2 - 3x_3 + 2x_1^2 - 1x_1x_3 + 0.2x_2^2 + 2x_3^2
	//
	// subject to x_1 + x_3 <= 2; x>=0
	for i := 0; i < 20; i++ {
		m := mip.NewModel()
		x1 := m.NewFloat(0, math.MaxFloat64)
		x2 := m.NewFloat(0, math.MaxFloat64)
		x3 := m.NewFloat(0, math.MaxFloat64)

		obj := m.Objective()
		obj.SetMinimize()
		obj.NewTerm(-1.0, x2)
		obj.NewTerm(-3.0, x3)
		obj.NewQuadraticTerm(2.0, x1, x1)
		obj.NewQuadraticTerm(-1.0, x1, x3)
		obj.NewQuadraticTerm(0.2, x2, x2)
		obj.NewQuadraticTerm(2.0, x3, x3)

		if !obj.IsQuadratic() {
			t.Error("Objective function not quadratic")
		}

		cstr := m.NewConstraint(mip.LessThanOrEqual, 2.0)
		cstr.NewTerm(1.0, x1)
		cstr.NewTerm(1.0, x3)

		solver, err := s(m)
		if err != nil {
			t.Errorf("Want nil, got %v", err)
		}
		options := defaultOptions()
		solution, err := solver.Solve(options)
		if err != nil {
			t.Errorf("Want nil, got %v", err)
		}
		if !solution.HasValues() {
			t.Errorf("expected to have a solution," +
				" solution hasValues returns false")
		}
		objVal := solution.ObjectiveValue()
		if math.Abs(objVal+2.45) > 0.001 {
			t.Errorf("got %v, want ~-2.45", objVal)
		}
	}
}

func qpTest2(s solverFactory, t *testing.T) {
	// minimize -0.5x1 + 2x_1^2
	//
	// subject to x1>=4
	m := mip.NewModel()
	x1 := m.NewFloat(4, math.MaxFloat64)

	obj := m.Objective()
	obj.SetMinimize()
	obj.NewQuadraticTerm(2.0, x1, x1)
	obj.NewTerm(-0.5, x1)
	if !obj.IsQuadratic() {
		t.Error("Objective function not quadratic")
	}

	solver, err := s(m)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	if !solution.HasValues() {
		t.Errorf("expected to have a solution," +
			" solution hasValues returns false")
	}
	objVal := solution.ObjectiveValue()
	if math.Abs(objVal-30.0) > 0.001 {
		t.Errorf("got %v, want ~30.0", objVal)
	}
}

func qpTest3(s solverFactory, t *testing.T) {
	// minimize x1 + x_2^2
	//
	// subject to x>=4
	m := mip.NewModel()
	x1 := m.NewFloat(4, math.MaxFloat64)
	x2 := m.NewFloat(4, math.MaxFloat64)

	obj := m.Objective()
	obj.SetMinimize()
	obj.NewQuadraticTerm(1.0, x2, x2)
	obj.NewTerm(1, x1)
	if !obj.IsQuadratic() {
		t.Error("Objective function not quadratic")
	}

	solver, err := s(m)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	if !solution.HasValues() {
		t.Errorf("expected to have a solution," +
			" solution hasValues returns false")
	}
	objVal := solution.ObjectiveValue()
	if math.Abs(objVal-20.0) > 0.001 {
		t.Errorf("got %v, want ~20.0", objVal)
	}
}

func linearRegression(s solverFactory, t *testing.T) {
	// some training data
	ys := []float64{1, 2, 3, 4, 5}
	xs := make([]float64, 0, len(ys))
	for _, y := range ys {
		xs = append(xs, y/2.0)
	}
	m := mip.NewModel()
	x := m.NewFloat(-math.MaxFloat64, math.MaxFloat64)

	obj := m.Objective()
	obj.SetMinimize()
	for i := range xs {
		// (xs[i] * x - ys[i])^2
		// <=> xs[i]^2 * x^2 - 2.0 * ys[i] * xs[i] * x + ys[i]^2
		obj.NewQuadraticTerm(xs[i]*xs[i], x, x)
		obj.NewTerm(-2.0*xs[i]*ys[i], x)
	}
	if !obj.IsQuadratic() {
		t.Error("Objective function not quadratic")
	}

	solver, err := s(m)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	if !solution.HasValues() {
		t.Errorf("expected to have a solution," +
			" solution hasValues returns false")
	}
	val := solution.Value(x)
	if math.Abs(val-2.0) > 0.001 {
		t.Errorf("Variable coefficient should be 2, got %v", val)
	}
}

func qpPortfolioOptim(s solverFactory, t *testing.T) {
	// Rebuilds and tests the model from
	// https://jump.dev/JuMP.jl/stable/tutorials/nonlinear/portfolio/

	m := mip.NewModel()
	obj := m.Objective()
	x := m.NewFloat(0, 1000)
	y := m.NewFloat(0, 1000)
	z := m.NewFloat(0, 1000)
	obj.NewQuadraticTerm(0.018641039983891217, x, x)
	obj.NewQuadraticTerm(0.0035985329276768114, x, y)
	obj.NewQuadraticTerm(0.0013097592536597557, x, z)
	obj.NewQuadraticTerm(0.0035985329276768114, y, x)
	obj.NewQuadraticTerm(0.0064369383226761, y, y)
	obj.NewQuadraticTerm(0.00488726515840726, y, z)
	obj.NewQuadraticTerm(0.0013097592536597557, z, x)
	obj.NewQuadraticTerm(0.00488726515840726, z, y)
	obj.NewQuadraticTerm(0.06868276545481435, z, z)
	obj.SetMinimize()

	c1 := m.NewConstraint(mip.GreaterThanOrEqual, 50.0)
	c1.NewTerm(0.026002150277777348, x)
	c1.NewTerm(0.008101316405671457, y)
	c1.NewTerm(0.0737159094919898, z)

	c2 := m.NewConstraint(mip.LessThanOrEqual, 1000.0)
	c2.NewTerm(1, x)
	c2.NewTerm(1, y)
	c2.NewTerm(1, z)

	solver, err := s(m)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}

	if int(solution.ObjectiveValue()) != 22634 {
		t.Errorf("Want ~22634, got %v", solution.ObjectiveValue())
	}

	if int(math.Round(solution.Value(x))) != 497 {
		t.Errorf("Want ~497, got %v", solution.Value(x))
	}

	if int(math.Round(solution.Value(y))) != 0 {
		t.Errorf("Want ~0, got %v", solution.Value(y))
	}

	if int(math.Round(solution.Value(z))) != 503 {
		t.Errorf("Want ~503, got %v", solution.Value(z))
	}
}

func qpAnotherExample(s solverFactory, t *testing.T) {
	// based on this example
	// https://rdrr.io/cran/ROI.plugin.quadprog/man/Example_01.html
	m := mip.NewModel()
	obj := m.Objective()
	x := m.NewFloat(0, math.MaxFloat64)
	y := m.NewFloat(0, math.MaxFloat64)
	z := m.NewFloat(0, math.MaxFloat64)
	obj.NewQuadraticTerm(0.5, x, x)
	obj.NewQuadraticTerm(0.5, y, y)
	obj.NewQuadraticTerm(0.5, z, z)
	obj.NewTerm(-5.0, y)
	obj.SetMinimize()

	c2 := m.NewConstraint(mip.GreaterThanOrEqual, -8)
	c2.NewTerm(-4, x)
	c2.NewTerm(-3, y)
	c2.NewTerm(0, z)

	c3 := m.NewConstraint(mip.GreaterThanOrEqual, 2)
	c3.NewTerm(2, x)
	c3.NewTerm(1, y)
	c2.NewTerm(0, z)

	c4 := m.NewConstraint(mip.GreaterThanOrEqual, 0)
	c4.NewTerm(0, x)
	c4.NewTerm(-2, y)
	c4.NewTerm(1, z)
	solver, err := s(m)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	options := defaultOptions()
	solution, err := solver.Solve(options)
	if err != nil {
		t.Errorf("Want nil, got %v", err)
	}
	if math.Abs(solution.ObjectiveValue()+2.380952e+00) > 0.01 {
		t.Errorf("Want ~-2.380952e+00, got %v", solution.ObjectiveValue())
	}
	if math.Abs(solution.Value(x)-0.4761905) > 0.01 {
		t.Errorf("Want ~0.4761905, got %v", solution.Value(x))
	}
	if math.Abs(solution.Value(y)-1.0476190) > 0.01 {
		t.Errorf("Want ~1.0476190, got %v", solution.Value(y))
	}
	if math.Abs(solution.Value(z)-2.0952381) > 0.01 {
		t.Errorf("Want ~2.0952381, got %v", solution.Value(z))
	}
}
