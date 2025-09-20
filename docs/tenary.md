# Balanced ternary notation

Balanced ternary is a numeral system that uses three digits that represent zero,
one and minus one. In Monogram these are `0`, `1`, and `T`. Here, `T` represents
`-1`, making it a "balanced" system because the digits are symmetric around
zero. This system is useful in certain mathematical and computational contexts
due to its ability to represent negative values without requiring a separate
sign.

- This is introduced with the prefix 0t and uses the digits 0, 1 and T. 
- In this notation T stands for -1 and the radix remains 3 as expected. 
- Like other numeric literals it supports both floating point and exponents.
- And like other numeric literals it supports underscores in the mantissa and
  fraction parts (only).

## Representation as a token

- Use the decimalradix "3"
- But add the field "balanced"=true.

## Balanced Ternary Digits

- `0`: Represents zero.
- `1`: Represents positive one.
- `T`: Represents negative one (`-1`).

## Examples of Balanced Ternary Integers

Here are some examples of integers represented in balanced ternary:

- `0t10` = `1 × 3^1 + 0 × 3^0 = 3`
- `0tT1` = `T × 3^1 + 1 × 3^0 = -3 + 1 = -2`
- `0t1T0` = `1 × 3^2 + T × 3^1 + 0 × 3^0 = 9 - 3 = 6`

## Examples of Balanced Ternary Fractions

Balanced ternary can also represent fractional values. Here are some examples:

- `0t.1` = `1 × 3^-1 = 1/3`
- `0t1.T`= `1 × 3^0 + T × 3^-1 = 1 - 1/3 = 2/3`
- `0tT.01` = `T × 3^0 + 0 × 3^-1 + 1 × 3^-2 = -1 + 1/9 = -8/9`

## Example of Balanced Ternary with Exponents

- `0tTTe-2` = `(T x 3^1 + T x 3^0) x 3^-2 = -4/9`

Note that the exponents are written in familiar decimal notation.

