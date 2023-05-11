import { Pipe, PipeTransform } from '@angular/core';
import { Container, ContainerGroup } from 'src/shared/container';

@Pipe({
  name: 'groupFilter',
})
export class GroupFilterPipe implements PipeTransform {
  private filter(group: ContainerGroup, filterStr: string): boolean {
    let matchName =
      group.project &&
      group.project.toLocaleLowerCase().includes(filterStr.toLocaleLowerCase());
    let matchContainerName = group.containers.some(container =>
      container.image
        .toLocaleLowerCase()
        .includes(filterStr.toLocaleLowerCase())
    );
    return matchName || matchContainerName;
  }

  transform(groups: ContainerGroup[], filterStr: string): ContainerGroup[] {
    if (!groups) return [];

    return groups.filter(containerGroup =>
      this.filter(containerGroup, filterStr)
    );
  }
}
