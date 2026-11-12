// Package ti86 implements ticalc.Calculator for the TI-86: reading and
// writing *.86? variable files and converting tokenized TI-BASIC programs to
// Unicode text and back.
//
// Importing the package registers the TI-86 in the ticalc registry:
//
//	import _ "github.com/hjkoskel/hkoTiUtil/ti86"
//
// Format reference: docs/ti86prot.txt (Tim Singer, "TI-86 Link Protocol and
// Variable Format Guide").
package ti86

import "github.com/hjkoskel/hkoTiUtil/ticalc"

// TI-86 variable type codes.
const (
	TypeReal             ticalc.VarType = 0x00
	TypeComplex          ticalc.VarType = 0x01
	TypeRealVector       ticalc.VarType = 0x02
	TypeComplexVector    ticalc.VarType = 0x03
	TypeRealList         ticalc.VarType = 0x04
	TypeComplexList      ticalc.VarType = 0x05
	TypeRealMatrix       ticalc.VarType = 0x06
	TypeComplexMatrix    ticalc.VarType = 0x07
	TypeRealConstant     ticalc.VarType = 0x08
	TypeComplexConstant  ticalc.VarType = 0x09
	TypeEquation         ticalc.VarType = 0x0A
	TypeString           ticalc.VarType = 0x0C
	TypeFunctionGDB      ticalc.VarType = 0x0D
	TypePolarGDB         ticalc.VarType = 0x0E
	TypeParametricGDB    ticalc.VarType = 0x0F
	TypeDiffEqGDB        ticalc.VarType = 0x10
	TypePicture          ticalc.VarType = 0x11
	TypeProgram          ticalc.VarType = 0x12
	TypeDirectory        ticalc.VarType = 0x15
	TypeFunctionWindow   ticalc.VarType = 0x17
	TypePolarWindow      ticalc.VarType = 0x18
	TypeParametricWindow ticalc.VarType = 0x19
	TypeDiffEqWindow     ticalc.VarType = 0x1A
	TypeSavedWindow      ticalc.VarType = 0x1B
	TypeBackup           ticalc.VarType = 0x1D
	TypeUnknown          ticalc.VarType = 0x1E
)

var typeInfos = map[ticalc.VarType]ticalc.TypeInfo{
	TypeReal:             {Name: "Real", Ext: "86n"},
	TypeComplex:          {Name: "Complex", Ext: "86c"},
	TypeRealVector:       {Name: "Real Vector", Ext: "86v"},
	TypeComplexVector:    {Name: "Complex Vector", Ext: "86v"},
	TypeRealList:         {Name: "Real List", Ext: "86l"},
	TypeComplexList:      {Name: "Complex List", Ext: "86l"},
	TypeRealMatrix:       {Name: "Real Matrix", Ext: "86m"},
	TypeComplexMatrix:    {Name: "Complex Matrix", Ext: "86m"},
	TypeRealConstant:     {Name: "Real Constant", Ext: "86k"},
	TypeComplexConstant:  {Name: "Complex Constant", Ext: "86k"},
	TypeEquation:         {Name: "Equation", Ext: "86e", Text: true},
	TypeString:           {Name: "String", Ext: "86s", Text: true},
	TypeFunctionGDB:      {Name: "Function GDB", Ext: "86d"},
	TypePolarGDB:         {Name: "Polar GDB", Ext: "86d"},
	TypeParametricGDB:    {Name: "Parametric GDB", Ext: "86d"},
	TypeDiffEqGDB:        {Name: "Differential Equation GDB", Ext: "86d"},
	TypePicture:          {Name: "Picture", Ext: "86i"},
	TypeProgram:          {Name: "Program", Ext: "86p", Text: true, Program: true},
	TypeDirectory:        {Name: "Directory"},
	TypeFunctionWindow:   {Name: "Function Window Settings", Ext: "86w"},
	TypePolarWindow:      {Name: "Polar Window Settings", Ext: "86w"},
	TypeParametricWindow: {Name: "Parametric Window Settings", Ext: "86w"},
	TypeDiffEqWindow:     {Name: "Differential Equation Window Settings", Ext: "86w"},
	TypeSavedWindow:      {Name: "Saved Window Size (ZRCL)", Ext: "86w"},
	TypeBackup:           {Name: "Memory Backup", Ext: "86b"},
	TypeUnknown:          {Name: "Unknown"},
}
