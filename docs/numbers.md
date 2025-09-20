# Number syntax

## Basic decimal numbers

The basic set of numbers are:

- positive and negative integers, e.g. 127, -1078, 0
- decimal fractions, e.g. 3.1415
- decimal floats (scientific notation) e.g. 7.2973525643e-3

## Alternative radixes

It is also possible to write these in other bases apart from base 10.

- Hex, binary and octal can be written with the generally familiar 0x, 0b and 0o prefixes. e.g. 0b1010, 0xFF, 0o127
- Alternatively the r for radix indicator can be used to express any base from 2 to 36. e.g. 2r1010, 16:FF, 8:127, 36rZ.
  - Bases greater than 16 simply use more upper-case letters of the alphabet. e.g. 36rHELLO = 29234652.
- It is possible to write floating point numbers in these non-decimal bases.
  - For example 0x1.1e2 = 272.0.
- Important: note that both the radix part and the exponent part are written in decimal notation.
- Important: note that the radix-marking character (x, o, b, r) must be lower
  case; the letters acting as digits are upper-case; the exponent marker (e)
  must be lower case. This is needed to cleanly separate the markers from the
  digits.

## Underscores

Nutmeg uses a very similar rule to Python: underscores may be used between 
any two characters of the mantissa or any two characters of the fraction.

### Represetation as a token

To assist the parser, which is the next stage in the compilation pipeline, we 
split the token up into:

- decimalradix, the radix in decimal notation (int n, 2 <= n <= 36)
- mantissa, the digits (or upper case letters) before the decimal point
- fraction, the digits (or upper case letters) after the decimal point
- exponent, the radix in decimal notation (int)

