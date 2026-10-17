(ns com.rpl.agent-o-rama.impl.types
  (:use [com.rpl.rama.path])
  (:require
   [clojure.set :as set]
   [clojure.string :as str]
   [com.rpl.agent-o-rama.impl.helpers :as h]
   [com.rpl.agent-o-rama.impl.serialize]
   [com.rpl.ramaspecter.defrecord-plus :as drp]
   [rpl.schema.core :as s])
  (:import
   [com.rpl.agentorama
    AgentComplete
    AgentInvoke
    AgentRef
    ExampleRun
    HumanInputRequest
    NestedOpType
    NodeInvoke
    RunInfo
    RunType
    ToolInfo]
   [com.rpl.agentorama.analytics
    AgentInvokeStats
    BasicAgentInvokeStats
    Feedback
    NestedOpInfo
    SubagentInvokeStats
    OpStats]
   [com.rpl.agentorama.source
    ActionSource
    AgentRunSource
    AiSource
    ApiSource
    BulkUploadSource
    EvalSource
    ExperimentSource
    HumanSource
    InfoSource]
   [com.rpl.agentorama.impl
    NippyMap]
   [com.rpl.rama.integration
    TaskGlobalObject]
   [dev.langchain4j.agent.tool
    ToolSpecification]
   [java.util
    UUID]
   [java.util.concurrent
    CompletableFuture]))

(defprotocol EvalTarget)
