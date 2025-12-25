# Sprint-Based Development Using a Documentation and a Coding Agent

## Premise 

LLMs are fantastics tools that can help us code at lighting fast speeds. But with our current limitations, they havea defining flaw. An LLM's context window is not capable of analyzing entire projects by itself.

One solution I have integrated for this problem is to simply delegate different tasks of a project's development to different LLM 'agents'. For the development of this project, I assign one LLM as the documentation authority, another as a coding authority, and then I use a third as a planning authority. This way, I am overloading one agent with too much context that will cause it to make mistakes.

The key to this system is to ensure that none of the agents, but particularly the documentation and the coding agents, never interact with context that belongs to another agent. This allows them to efficiently perform their tasks without being overloaded with context that is irrelevant to their goals.

## Workflow

I have integrated the following workflow when completing a 'sprint' as I develop this project.

### Phase 1 - Planning

Before even considering developing a sprint, I must plan out its objective, requirements, deliverables, etc. Defining a sprint is key to its success. A well defined sprint leads to implementing a feature or change that will meet all the expected criteria.

To plan out a sprint, I use the planning agent. I go on chat.com or any other LLM on the internet. I state my goal to the LLM clearly, and task with with creating well defined planning documents. These are `planning.md`, `spec.md`, `tdd.md`, and `data-dictionary.md`. These documents provide enough information for the documentation agent about our project, without overloading it with context that can lead it to make mistakes.

### Phase 2 - Plan Assesment

When our planning documents are completed, we then move onto an assessment of these documents. In this phase, we must ask the documentation agent to review the feature/change implementation plan and report any inconsistencies with the logic, or in general anything that needs clarification. 

After review, the documentation agent must decide if the planning documents are **approved**, **approved with required clarification**, or **not approved**, meaning that there are logic gaps in the planning stage that must be addressed.

Depending on the documentation agent's review, we must decide whether to move onto phase 3, or to make adjustments to our implementation plan.

### Phase 3 - Implementation

If the planning documentation is approved, our task now is to copy the `spec.md` and `tdd.md` documents from `docs/sprints/sprint-##/planning/` to `docs/sprints/sprint-##/in-process/implementation-engine/`. After we must ask the documentation agent to make a series of prompts to feed the coding agent with for the implementation of the sprint. These prompts must reinforce that the coding agent's authority are the planning docs in `in-process/implementation-engine/`. That way, its context is not overloaded.

We must also create prompts that tell the coding agent to document its implementation in `in-process/implementation-engine/notes.md`. This way the coding agent can log anything it faced during its implementation of the feature/change.

### Phase 4 - Implementaion Assesment

After the implementation phase, we must go back to the documentation agent and tell it to assess the coding agent's implementation. The documentation agent must read `notes.md` file created by the coding agent and find any inconsistencies or implementation concerns with the coding agent's implementation. In this phase, we tell the documentation agent to analyze the coding agent's notes with the planning documentation, and assess whether the implementation matches with the goals of the planning documents and their specifications.

The documentation agent must then decide whether the implementation is **approved**, **approved with required changes** (often clarification, and not particularly associated with changin the codebase), and **not approved** due to a gap in logic or raised concerns associated with the implementation. If the implementation is not approved, we must gather any information from the approval status, and return to **phase 3 - implementation**.

### Phase 5 - Sprint Closure

After the implementation has been approved, we must use the documentation agent to close out and freeze the sprint. We must tell this agent to populate the `backlog/sprint-##.md` and `completed/sprint-##-summary.md` files. The summary file must make a brief of the process of implementing this sprint (goals, challenges faced, outcomes, etc). The backlog file for this sprint will list any deferred features, known limitations, and future considerations associated with the sprint. This file can be used by the planning agent for a subsequent sprint.

## Conclusion

This process has been effective at making realiable projects. Following this development guide will allow developers to implement changes using LLMs responsibly and efficiently.