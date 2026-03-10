export type TaskStatus =
  | 'QUEUED'
  | 'PROCESSING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TOO_LARGE'
  | 'NO_PROVIDER'

export type CommandType = 'plan' | 'execute' | 'auto'

export interface Task {
  ID: string
  ProjectPath: string
  TargetFile: string
  Instruction: string
  ContextFiles: string[]
  ModelID: string
  ProviderHint: string
  Command: CommandType
  Status: TaskStatus
  CreatedAt: string
  UpdatedAt: string
  Logs: string
}

export interface TaskInput {
  ProjectPath: string
  TargetFile: string
  Instruction: string
  ContextFiles?: string[]
  ModelID?: string
  ProviderHint?: string
  Command?: CommandType
}

export interface ProviderInfo {
  Name: string
  Active: boolean
  ActiveModel: string
  Models: string[]
}
