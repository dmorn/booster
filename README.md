# booster

## The Idea
Look around you. How many devices capable of connecting (autonomously) to the Internet are there? We're used to exploit just one of them per time, even if the workload is very high (for example when using a streaming service or exchanging large files).

`Booster` wants to overcome this situation by splitting the requests (when possible) into chunks, each one representing a job that can be parallelized and performed by an **helper** device. Each **helper** performs the request that `booster` is asking for, collects the data and sends it back.

Everything is made transparent using a custom **proxy** that the user must setup and connect to in order to let `booster` know which requests can be.. boosted 😁☄️
