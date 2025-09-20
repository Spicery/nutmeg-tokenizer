# Recognising newlines

Nutmeg allows newlines to act as end-of-statement markers. To implement this, we
simpy record whether a token was preceded by a newline and/or followed by a
newline. Strictly speaking it is any separating whitespace-and-comments that
include a newline that counts.

To indicate this, tokens have two boolean attributes ln_before and ln_after.
