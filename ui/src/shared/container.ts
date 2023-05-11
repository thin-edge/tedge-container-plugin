export interface Container {
  id: string;
  name: string;
  containerId: string;
  ports: string;
  command: string;
  networks: string;
  filesystem: string;
  image: string;
  runningFor: string;
  state: string;
  status: string;
  project?: string;
  lastUpdated: string;
}

export interface ContainerGroup {
  project: string;
  containers: Container[];
}

export interface ContainerParent {
  name: string;
  id: string | number;
}
