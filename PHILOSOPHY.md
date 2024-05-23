# How Sequencer is built

What I want to do here is to explain the the philosophy that I want to adopt for this project so that if someone wants to understand the reason why a certain feature is built the way it is, they can find some explanation here (hopefully). There exists many different ways to approach software engineering, and there's no True Way of working in this field, so while this is _a_ way, it's not the only way, and it may not be the best way out there, but it's the one I feel the most comfortable with.

## When it comes to any feature: Quality > Quantity

This might not be very controversial, but I think it's worth talking about. Sequencer is a cloud native project that can run pretty much on any stack. This means that there's going to be environments where Sequencer might not work very well. Whether it's a lack of support for a specific feature of a certain cloud provider, or just some permission conflict. When it comes to features for Sequencer, the goal is to always aim for feature stability. Stability doesn't means "it never errors", 

## Expose errors back to the user, with meaningful information

Every error that cannot be safely handled independently of the user needs to be surfaced back to the user. Errors should start with `E#[NUMBER]` where the number is a unique number that links to a more thorough description of what the error is, and what kind of solution the user might be looking for. Those reference number are listed on the [error's](./docs/errors.md) page and should be kept up to date with the main branch.

## Duplication is cheaper than the wrong abstraction

Abstractions are often "obvious" when written, but often don't pass the test of time. What seems like a "simple" abstraction at the time can become a roadblock for future improvements down the road. When looking to abstract a part of the code away into a shared function, this questions can help find if the is a valid usecase for an abstraction:

1. Is the code I'm abstraction 100% the same (99% is not close enough)?
2. Are they conceptually doing the exact same thing?
3. Does the abstraction make the code harder to reason about?
4. Is the intent still clear with arguments/options?

Code duplication, while adding more line of code to the project, is usually easier to reason about for someone who reads the codebase. In many cases, code duplication is also faster due to less branching/conditions.

With all of this being said, abstractions do need to exist in this codebase. It's all a balance!


Pier-Olivier