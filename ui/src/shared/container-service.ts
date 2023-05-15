import { Injectable } from '@angular/core';
import { ContainerGroup, Container, ContainerParent } from './container';
import { IManagedObject, IResultList, InventoryService } from '@c8y/client';
@Injectable({
  providedIn: 'root',
})
export class ContainerService {
  constructor(private inventory: InventoryService) {}

  async getContainers(device: string): Promise<Container[]> {
    const filter = {
      query: 'serviceType eq container or serviceType eq container-group',
      pageSize: 100,
      withTotalPages: true,
    };
    return this.inventory
      .childAdditionsList(device, filter)
      .then(res => res.data.map(mo => this.managedObjectToContainer(mo)));
  }



  async getContainerGroups(device: string): Promise<ContainerGroup[]> {
    return this.getContainers(device).then(res =>
      this.containerToContainerGroups(res)
    );
  }

  async getContainer(serviceId: string): Promise<[Container, ContainerParent]> {
    const filter = {
      withParents: true,
    };
    return this.inventory
      .detail(serviceId, filter)
      .then(res => this.managedObjectToContainerWithParent(res.data));
  }

  stop(container: Container) {
    console.log(
      'Stopping Container' +
        container.containerId +
        ', unfortnunatley it is not implementet yet'
    );
  }

  private managedObjectToContainer(mo: IManagedObject): Container {
    return {
      id: mo.id,
      name: mo.name,
      containerId: mo.containerId,
      ports: mo.ports,
      command: mo.command,
      networks: mo.networks,
      filesystem: mo.filesystem,
      image: mo.image,
      runningFor: mo.runningFor,
      state: mo.state,
      status: mo.status,
      project: mo.projectName,
      lastUpdated: mo.lastUpdated,
    };
  }
  private managedObjectToContainerWithParent(
    mo: IManagedObject
  ): [Container, ContainerParent] {
    let parent = mo.additionParents.references.pop().managedObject;
    return [
      this.managedObjectToContainer(mo),
      { name: parent?.name, id: parent?.id },
    ];
  }

  private containerToContainerGroups(
    containers: Container[]
  ): ContainerGroup[] {
    let projects: string[] = containers
      .map(c => c.project)
      .filter((value, index, array) => array.indexOf(value) === index && value);
    return projects.map(p => {
      return { project: p, containers: containers.filter(c => c.project == p) };
    });
  }
}
