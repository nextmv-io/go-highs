// © 2019-present nextmv.io inc

// Package highs implements HiGHS solver bindings.
package highs

/*
   #cgo darwin,arm64 LDFLAGS: ${SRCDIR}/external/darwin-arm64/lib/libhighs.a -lc++
   #cgo darwin,arm64 CFLAGS: -I${SRCDIR}/external/darwin-arm64/include/highs -mmacosx-version-min=11.0
   #cgo darwin,amd64 LDFLAGS: ${SRCDIR}/external/darwin-amd64/lib/libhighs.a -lc++
   #cgo darwin,amd64 CFLAGS: -I${SRCDIR}/external/darwin-amd64/include/highs -mmacosx-version-min=11.0
   #cgo linux,amd64 LDFLAGS: ${SRCDIR}/external/linux-amd64/lib/libhighs.a -lstdc++ -lm -ldl -lz
   #cgo linux,amd64 CFLAGS: -I${SRCDIR}/external/linux-amd64/include/highs
   #cgo linux,arm64 LDFLAGS: ${SRCDIR}/external/linux-arm64/lib/libhighs.a -lstdc++ -lm -ldl
   #cgo linux,arm64 CFLAGS: -I${SRCDIR}/external/linux-arm64/include/highs
   #cgo CXXFLAGS: -std=c++11
   #include "interfaces/highs_c_api.h"
   #include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"sort"
	"time"
	"unsafe"

	"github.com/nextmv-io/go-mip"
)

// on Windows/WSL ubuntu amd64 we currently have to link against zlib
// dynamically. (this comment needs to come below the import "C" statement)

// NewSolver creates solver using Highs as back-end solver.
func NewSolver(model mip.Model) mip.Solver {
	return &solverHighs{
		model: model,
	}
}

// Solve solves a given model with some options.
func (solver *solverHighs) Solve(options mip.SolveOptions) (mip.Solution, error) {
	start := time.Now()
	if len(solver.model.Vars()) == 0 {
		return &highsSolution{
			solutionStatus: optimal,
		}, nil
	}

	highsPtr := C.Highs_create()
	if highsPtr == nil {
		return &highsSolution{
			solutionStatus: statusUnknown,
		}, nil
	}
	defer C.Highs_destroy(highsPtr)

	input := solver.newHighsInput(highsPtr, start)

	if err := handleOptions(highsPtr, *input, options); err != nil {
		return nil, fmt.Errorf("error handling options in HiGHS solver: %w", err)
	}

	isMiqp := solver.model.Objective().IsQuadratic() &&
		input.isIntegerProblem
	if isMiqp {
		return nil, errMiqpNotSupported
	}

	return solve(highsPtr, options, input)
}

type highsSolution struct {
	values         []float64
	solutionStatus solutionStatus
	objectiveValue float64
	runtime        time.Duration
}

func (l *highsSolution) ObjectiveValue() float64 {
	return l.objectiveValue
}

func (l *highsSolution) Provider() mip.SolverProvider {
	return "HiGHS"
}

func (l *highsSolution) RunTime() time.Duration {
	return l.runtime
}

func (l *highsSolution) Value(variable mip.Var) float64 {
	if variable.Index() >= len(l.values) {
		return math.MaxFloat64
	}

	return l.values[variable.Index()]
}

func (l *highsSolution) IsNumericalFailure() bool {
	return false
}

func (l *highsSolution) IsOptimal() bool {
	return l.solutionStatus == optimal
}

func (l *highsSolution) HasValues() bool {
	return hasValues(l.solutionStatus)
}

func (l *highsSolution) IsSubOptimal() bool {
	return false
}

func (l *highsSolution) IsTimeOut() bool {
	return l.solutionStatus == timeLimit
}

func (l *highsSolution) IsUnbounded() bool {
	return l.solutionStatus == unbounded ||
		l.solutionStatus == unboundedOrInfeasible
}

func (l *highsSolution) IsInfeasible() bool {
	return l.solutionStatus == infeasible ||
		l.solutionStatus == unboundedOrInfeasible
}

type solverHighs struct {
	model mip.Model
}

type highsInput struct {
	start                      time.Time
	rowUpperBound              []C.double
	rowLowerBound              []C.double
	columnCosts                []C.double
	columnLowerBound           []C.double
	columnUpperBound           []C.double
	rowConstraintMatrixIndices []C.int
	rowConstraintMatrixBegins  []C.int
	rowConstraintMatrixValues  []C.double
	hessianMatrixIndices       []C.int
	hessianMatrixBegins        []C.int
	hessianMatrixValues        []C.double
	columnIntegrality          []C.int
	numNonZeros                int
	numQuadraticNonZeros       int
	numColumns                 int
	numRows                    int
	sense                      C.int
	isIntegerProblem           bool
	isQuadraticProblem         bool
}

type solutionStatus int

// the values match HiGHS internal codes
const (
	optimal               solutionStatus = 7
	infeasible            solutionStatus = 8
	unboundedOrInfeasible solutionStatus = 9
	unbounded             solutionStatus = 10
	timeLimit             solutionStatus = 13
	statusUnknown         solutionStatus = 15
)

func hasValues(solutionStatus solutionStatus) bool {
	return solutionStatus == optimal
}

func (solver *solverHighs) newHighsInput(
	highsPtr unsafe.Pointer,
	start time.Time,
) *highsInput {
	// infinity is defined as
	// std::numeric_limits<double>::infinity()
	// by HiGHS.
	infinity := C.Highs_getInfinity(highsPtr)

	input := new(highsInput)
	input.start = start

	input.numColumns = len(solver.model.Vars())
	allConstraints := solver.model.Constraints()
	constraintsWithTerms := make(mip.Constraints, 0, len(allConstraints))
	for _, c := range allConstraints {
		if len(c.Terms()) > 0 {
			constraintsWithTerms = append(constraintsWithTerms, c)
		}
	}

	input.numRows = len(constraintsWithTerms)

	prepareColumns(input, solver, constraintsWithTerms, infinity)

	prepareConstraintMatrix(input, solver, constraintsWithTerms, infinity)

	prepareHessian(input, solver)
	input.isQuadraticProblem = input.numQuadraticNonZeros > 0

	input.sense = C.kHighsObjSenseMinimize

	if solver.model.Objective().IsMaximize() {
		input.sense = C.kHighsObjSenseMaximize
	}
	return input
}

func mapVarTypeToIntegrality(variable mip.Var) C.int {
	if variable.IsBool() || variable.IsInt() {
		return C.kHighsVarTypeInteger
	}

	return C.kHighsVarTypeContinuous
}

func handleOptions(highsPtr unsafe.Pointer, input highsInput, options mip.SolveOptions) error {
	if err := setOutputFlag(highsPtr, options); err != nil {
		return err
	}

	if err := setDoubleOption(
		highsPtr,
		"time_limit",
		options.Duration.Seconds(),
	); err != nil {
		return err
	}

	if input.isIntegerProblem {
		if err := setDoubleOption(
			highsPtr,
			"mip_abs_gap",
			options.MIP.Gap.Absolute,
		); err != nil {
			return err
		}

		if err := setDoubleOption(
			highsPtr,
			"mip_rel_gap",
			options.MIP.Gap.Relative,
		); err != nil {
			return err
		}
	}

	controlOptions, err := options.Control.ToTyped()
	if err != nil {
		return err
	}

	for _, option := range controlOptions.Bool {
		if err := setBoolOption(
			highsPtr,
			option.Name,
			option.Value,
		); err != nil {
			return err
		}
	}

	for _, option := range controlOptions.Float {
		if err := setDoubleOption(
			highsPtr,
			option.Name,
			option.Value,
		); err != nil {
			return err
		}
	}

	for _, option := range controlOptions.Int {
		if err := setIntOption(
			highsPtr,
			option.Name,
			option.Value,
		); err != nil {
			return err
		}
	}

	for _, option := range controlOptions.String {
		if err := setStringOption(
			highsPtr,
			option.Name,
			option.Value,
		); err != nil {
			return err
		}
	}

	return nil
}

func setOutputFlag(ptr unsafe.Pointer, options mip.SolveOptions) error {
	option := C.CString("output_flag")
	defer C.free(unsafe.Pointer(option))
	verbosityLevel := C.int(1)
	if options.Verbosity == mip.Off {
		verbosityLevel = C.int(0)
	}
	status := C.Highs_setBoolOptionValue(ptr, option, verbosityLevel)
	if status != C.kHighsStatusOk {
		return errors.New(
			"highs failed setting verbosity level",
		)
	}

	reportLevel := C.int(0)

	switch options.Verbosity {
	case mip.Low:
		reportLevel = C.int(0)
	case mip.Medium:
		reportLevel = C.int(1)
	case mip.High:
		reportLevel = C.int(2)
	}
	option2 := C.CString("log_dev_level")
	defer C.free(unsafe.Pointer(option2))

	status = C.Highs_setIntOptionValue(
		ptr,
		option2,
		reportLevel,
	)

	if status != C.kHighsStatusOk {
		return errors.New(
			"highs failed setting verbosity mip report level",
		)
	}
	return nil
}

func setBoolOption(highsPtr unsafe.Pointer, option string, value bool) error {
	optionName := C.CString(option)
	defer C.free(unsafe.Pointer(optionName))

	cBool := C.int(0)
	if value {
		cBool = C.int(1)
	}

	status := C.Highs_setBoolOptionValue(highsPtr, optionName, cBool)
	if status != C.kHighsStatusOk {
		return fmt.Errorf("HiGHS failed setting bool option %s to value %v", option, value)
	}

	return nil
}

func setDoubleOption(highsPtr unsafe.Pointer, option string, value float64) error {
	optionName := C.CString(option)
	defer C.free(unsafe.Pointer(optionName))
	status := C.Highs_setDoubleOptionValue(highsPtr, optionName, C.double(value))
	if status != C.kHighsStatusOk {
		return fmt.Errorf("HiGHS failed setting float (double) option %s to value %v", option, value)
	}

	return nil
}

func setIntOption(highsPtr unsafe.Pointer, option string, value int) error {
	optionName := C.CString(option)
	defer C.free(unsafe.Pointer(optionName))
	status := C.Highs_setIntOptionValue(highsPtr, optionName, C.int(value))
	if status != C.kHighsStatusOk {
		return fmt.Errorf("HiGHS failed setting int option %s to value %v", option, value)
	}

	return nil
}

func setStringOption(highsPtr unsafe.Pointer, option string, value string) error {
	optionName := C.CString(option)
	defer C.free(unsafe.Pointer(optionName))
	optionValue := C.CString(value)
	defer C.free(unsafe.Pointer(optionValue))
	status := C.Highs_setStringOptionValue(highsPtr, optionName, optionValue)
	if status != C.kHighsStatusOk {
		return fmt.Errorf("HiGHS failed setting string option %s to value %v", option, value)
	}

	return nil
}

func solve(
	highsPtr unsafe.Pointer, _ mip.SolveOptions, input *highsInput,
) (*highsSolution, error) {
	pRowLowerBound := (*C.double)(unsafe.Pointer(nil))
	pRowUpperBound := (*C.double)(unsafe.Pointer(nil))
	pColumnIntegrality := (*C.int)(unsafe.Pointer(nil))
	pRowConstraintMatrixBegins := (*C.int)(unsafe.Pointer(nil))
	pRowConstraintMatrixIndices := (*C.int)(unsafe.Pointer(nil))
	pRowConstraintMatrixValues := (*C.double)(unsafe.Pointer(nil))
	pColumnCosts := (*C.double)(unsafe.Pointer(nil))
	pColumnLowerBound := (*C.double)(unsafe.Pointer(nil))
	pColumnUpperBound := (*C.double)(unsafe.Pointer(nil))
	pHessianConstraintMatrixBegins := (*C.int)(unsafe.Pointer(nil))
	pHessianConstraintMatrixIndices := (*C.int)(unsafe.Pointer(nil))
	pHessianConstraintMatrixValues := (*C.double)(unsafe.Pointer(nil))

	if input.numRows > 0 {
		pRowLowerBound = (*C.double)(unsafe.Pointer(&input.rowLowerBound[0]))
		pRowUpperBound = (*C.double)(unsafe.Pointer(&input.rowUpperBound[0]))
		b := (*C.int)(unsafe.Pointer(&input.rowConstraintMatrixBegins[0]))
		pRowConstraintMatrixBegins = b
		i := (*C.int)(unsafe.Pointer(&input.rowConstraintMatrixIndices[0]))
		pRowConstraintMatrixIndices = i
		v := (*C.double)(unsafe.Pointer(&input.rowConstraintMatrixValues[0]))
		pRowConstraintMatrixValues = v
	}

	if input.isQuadraticProblem {
		pHessianConstraintMatrixBegins = (*C.int)(unsafe.Pointer(
			&input.hessianMatrixBegins[0],
		))
		pHessianConstraintMatrixIndices = (*C.int)(unsafe.Pointer(
			&input.hessianMatrixIndices[0],
		))
		pHessianConstraintMatrixValues = (*C.double)(unsafe.Pointer(
			&input.hessianMatrixValues[0],
		))
	}

	if input.numColumns > 0 {
		pColumnCosts = (*C.double)(unsafe.Pointer(
			&input.columnCosts[0],
		))
		pColumnLowerBound = (*C.double)(unsafe.Pointer(
			&input.columnLowerBound[0],
		))
		pColumnUpperBound = (*C.double)(unsafe.Pointer(
			&input.columnUpperBound[0],
		))
	}

	if input.isIntegerProblem {
		pColumnIntegrality = (*C.int)(unsafe.Pointer(
			&input.columnIntegrality[0],
		))
	}

	status := C.Highs_passModel(
		highsPtr,
		C.int(input.numColumns),
		C.int(input.numRows),
		C.int(input.numNonZeros),
		C.int(input.numQuadraticNonZeros),
		C.kHighsMatrixFormatRowwise,
		C.kHighsHessianFormatTriangular,
		input.sense,
		C.double(0.0),
		pColumnCosts,
		pColumnLowerBound,
		pColumnUpperBound,
		pRowLowerBound,
		pRowUpperBound,
		pRowConstraintMatrixBegins,
		pRowConstraintMatrixIndices,
		pRowConstraintMatrixValues,
		pHessianConstraintMatrixBegins,
		pHessianConstraintMatrixIndices,
		pHessianConstraintMatrixValues,
		pColumnIntegrality,
	)

	if status != C.kHighsStatusOk {
		return &highsSolution{
			solutionStatus: statusUnknown,
		}, errPassing
	}

	runStatus := C.Highs_run(highsPtr)

	if !(runStatus == C.kHighsStatusOk || runStatus == C.kHighsStatusWarning) {
		return &highsSolution{
			solutionStatus: statusUnknown,
		}, nil
	}
	modelStatus := C.Highs_getModelStatus(highsPtr)

	columnValues := make([]float64, input.numColumns)
	columnDuals := make([]C.double, input.numColumns)

	rowValues := make([]float64, input.numRows)
	rowDuals := make([]C.double, input.numRows)

	pRowValues := (*C.double)(unsafe.Pointer(nil))
	pRowDuals := (*C.double)(unsafe.Pointer(nil))

	if input.numRows > 0 {
		pRowValues = (*C.double)(unsafe.Pointer(&rowValues[0]))
		pRowDuals = (*C.double)(unsafe.Pointer(&rowDuals[0]))
	}

	status = C.Highs_getSolution(
		highsPtr,
		(*C.double)(unsafe.Pointer(&columnValues[0])),
		(*C.double)(unsafe.Pointer(&columnDuals[0])),
		pRowValues,
		pRowDuals,
	)

	if status != C.kHighsStatusOk {
		return &highsSolution{
			solutionStatus: statusUnknown,
		}, errGetSolution
	}
	objectiveValue := float64(C.Highs_getObjectiveValue(highsPtr))
	runtime.KeepAlive(input)
	return &highsSolution{
		objectiveValue: objectiveValue,
		runtime:        time.Since(input.start),
		solutionStatus: solutionStatus(modelStatus),
		values:         columnValues,
	}, nil
}

var (
	errPassing = errors.New(
		"highs failed passing the model",
	)
	errGetSolution = errors.New(
		"highs failed getting the solution",
	)
	errMiqpNotSupported = errors.New(
		"highs does not support mixed integer quadratic programs",
	)
)

func prepareColumns(
	input *highsInput,
	solver *solverHighs,
	constraintsWithTerms mip.Constraints,
	_ C.double,
) {
	input.numNonZeros = 0
	for _, c := range constraintsWithTerms {
		input.numNonZeros += len(c.Terms())
	}

	input.columnCosts = make([]C.double, input.numColumns)
	input.columnLowerBound = make([]C.double, input.numColumns)
	input.columnUpperBound = make([]C.double, input.numColumns)
	input.columnIntegrality = make([]C.int, input.numColumns)

	input.isIntegerProblem = false

	for _, v := range solver.model.Vars() {
		i := v.Index()
		input.columnCosts[i] = C.double(0.0)
		input.columnLowerBound[i] = C.double(v.LowerBound())
		input.columnUpperBound[i] = C.double(v.UpperBound())
		t := mapVarTypeToIntegrality(v)
		input.columnIntegrality[i] = t
		if t == C.kHighsVarTypeInteger {
			input.isIntegerProblem = true
		}
	}

	for _, term := range solver.model.Objective().Terms() {
		input.columnCosts[term.Var().Index()] = C.double(term.Coefficient())
	}
}

func prepareConstraintMatrix(
	input *highsInput,
	_ *solverHighs,
	constraintsWithTerms mip.Constraints,
	infinity C.double,
) {
	input.rowConstraintMatrixBegins = make([]C.int, input.numRows)
	input.rowLowerBound = make([]C.double, input.numRows)
	input.rowUpperBound = make([]C.double, input.numRows)

	input.rowConstraintMatrixIndices = make([]C.int, input.numNonZeros)
	input.rowConstraintMatrixValues = make([]C.double, input.numNonZeros)

	rowConstraintMatrixBegin := 0

	for i, c := range constraintsWithTerms {
		input.rowConstraintMatrixBegins[i] = C.int(
			rowConstraintMatrixBegin,
		)
		rhs := C.double(c.RightHandSide())
		switch c.Sense() {
		case mip.LessThanOrEqual:
			input.rowLowerBound[i] = -infinity
			input.rowUpperBound[i] = rhs
		case mip.Equal:
			input.rowLowerBound[i] = rhs
			input.rowUpperBound[i] = rhs
		case mip.GreaterThanOrEqual:
			input.rowLowerBound[i] = rhs
			input.rowUpperBound[i] = infinity
		}

		for _, t := range c.Terms() {
			i := C.int(t.Var().Index())
			input.rowConstraintMatrixIndices[rowConstraintMatrixBegin] = i
			c := C.double(t.Coefficient())
			input.rowConstraintMatrixValues[rowConstraintMatrixBegin] = c

			rowConstraintMatrixBegin++
		}
	}
}

func prepareHessian(input *highsInput, solver *solverHighs) {
	qTerms := solver.model.Objective().QuadraticTerms()
	qMat := make(map[int]map[int]mip.QuadraticTerm)
	for _, v := range qTerms {
		_, ok := qMat[v.Var1().Index()]
		if !ok {
			qMat[v.Var1().Index()] = make(map[int]mip.QuadraticTerm)
		}
		qMat[v.Var1().Index()][v.Var2().Index()] = v
	}
	input.numQuadraticNonZeros = len(qTerms)
	input.hessianMatrixBegins = make([]C.int, 0, input.numColumns+1)
	input.hessianMatrixIndices = make([]C.int, 0, input.numQuadraticNonZeros)
	input.hessianMatrixValues = make([]C.double, 0, input.numQuadraticNonZeros)
	nonZeros := 0
	for i := 0; i < input.numColumns; i++ {
		input.hessianMatrixBegins = append(input.hessianMatrixBegins, C.int(nonZeros))
		row, ok := qMat[i]
		if !ok {
			continue
		}
		terms := make(mip.QuadraticTerms, 0, len(row))
		for _, v := range row {
			terms = append(terms, v)
		}
		sort.SliceStable(terms, func(i, j int) bool {
			return terms[i].Var2().Index() < terms[j].Var2().Index()
		})
		for _, v := range terms {
			input.hessianMatrixIndices = append(
				input.hessianMatrixIndices,
				C.int(v.Var2().Index()),
			)
			coef := v.Coefficient()
			if v.Var1().Index() == v.Var2().Index() {
				coef *= 2.0
			}
			// highs solves the problem of min/max c^tx * 1/2*x^tQx
			// however it is more intuitive (we assume) when developers
			// can expect min/max c^tx * x^tQx.
			input.hessianMatrixValues = append(
				input.hessianMatrixValues,
				C.double(coef),
			)
		}
		nonZeros += len(terms)
	}
	input.hessianMatrixBegins = append(
		input.hessianMatrixBegins,
		C.int(input.numQuadraticNonZeros),
	)
}
